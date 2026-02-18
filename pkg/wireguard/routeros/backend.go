package routeros

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

const (
	defaultHTTPSPort = "443"
	defaultRESTPath  = "/rest"
)

var errRouterOSResourceNotFound = errors.New("routeros resource not found")

type parsedURL struct {
	baseURL            string
	username           string
	password           string
	insecureSkipVerify bool
}

type routerOSBackend struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

type entry map[string]string

func Register() {
	driver.Register("routeros", func(_ context.Context, rawURL string) (driver.Backend, error) {
		return NewRouterOSBackend(rawURL)
	}, true)
}

func NewRouterOSBackend(rawURL string) (driver.Backend, error) {
	parsed, err := parseURL(rawURL)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: parsed.insecureSkipVerify, //nolint:gosec // explicitly user-controlled
		},
	}

	return &routerOSBackend{
		baseURL:  parsed.baseURL,
		username: parsed.username,
		password: parsed.password,
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}, nil
}

func (b *routerOSBackend) Device(ctx context.Context, name string) (*driver.Device, error) {
	iface, err := b.findWireguardInterfaceByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if iface == nil {
		return nil, fmt.Errorf("interface not found: %s", name)
	}

	addresses, err := b.interfaceAddresses(ctx, name)
	if err != nil {
		return nil, err
	}

	peers, err := b.interfacePeers(ctx, iface)
	if err != nil {
		return nil, err
	}

	return &driver.Device{
		Interface: driver.Interface{
			Name:      value(iface, "name"),
			Addresses: addresses,
			Mtu:       intValue(iface, "mtu"),
		},
		Wireguard: driver.Wireguard{
			Name:         value(iface, "name"),
			PublicKey:    value(iface, "public-key"),
			PrivateKey:   value(iface, "private-key"),
			ListenPort:   intValue(iface, "listen-port"),
			FirewallMark: intValue(iface, "firewall-mark", "fwmark"),
			Peers:        peers,
		},
	}, nil
}

func (b *routerOSBackend) Up(ctx context.Context, options driver.ConfigureOptions) (*driver.Device, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	name := options.InterfaceOptions.Name
	iface, err := b.findWireguardInterfaceByName(ctx, name)
	if err != nil {
		return nil, err
	}

	payload := map[string]string{
		"name":        name,
		"private-key": options.WireguardOptions.PrivateKey,
		"disabled":    "false",
		"comment":     options.InterfaceOptions.Description,
	}
	if options.InterfaceOptions.Mtu > 0 {
		payload["mtu"] = strconv.Itoa(options.InterfaceOptions.Mtu)
	}
	if options.WireguardOptions.ListenPort != nil {
		payload["listen-port"] = strconv.Itoa(*options.WireguardOptions.ListenPort)
	}

	if iface == nil {
		if err := b.putEntry(ctx, "interface/wireguard", payload); err != nil {
			return nil, err
		}
	} else {
		if interfaceNeedsPatch(iface, payload) {
			if err := b.patchEntry(ctx, "interface/wireguard", value(iface, ".id"), payload); err != nil {
				return nil, err
			}
		}
	}

	currentIface, err := b.findWireguardInterfaceByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if currentIface == nil {
		return nil, fmt.Errorf("failed to find interface after up: %s", name)
	}

	if err := b.reconcileAddresses(ctx, name, options.InterfaceOptions.Address); err != nil {
		return nil, err
	}
	if err := b.reconcilePeers(ctx, currentIface, options.WireguardOptions.Peers); err != nil {
		return nil, err
	}

	return b.Device(ctx, name)
}

func (b *routerOSBackend) Down(ctx context.Context, name string) error {
	iface, err := b.findWireguardInterfaceByName(ctx, name)
	if err != nil {
		return err
	}
	if iface == nil {
		return nil
	}

	peers, err := b.interfacePeerEntries(ctx, iface)
	if err != nil {
		return err
	}
	for _, p := range peers {
		// RouterOS dynamic peers are owned by other subsystems and cannot be edited/deleted.
		if boolValue(p, "dynamic") {
			continue
		}
		if err := b.deleteEntry(ctx, "interface/wireguard/peers", value(p, ".id")); err != nil {
			return err
		}
	}

	addressEntries, err := b.interfaceAddressEntries(ctx, name)
	if err != nil {
		return err
	}
	for _, address := range addressEntries {
		if err := b.deleteEntry(ctx, "ip/address", value(address, ".id")); err != nil {
			return err
		}
	}

	return b.deleteEntry(ctx, "interface/wireguard", value(iface, ".id"))
}

func (b *routerOSBackend) Status(ctx context.Context, name string) (bool, error) {
	iface, err := b.findWireguardInterfaceByName(ctx, name)
	if err != nil {
		return false, err
	}
	if iface == nil {
		return false, nil
	}

	disabled := boolValue(iface, "disabled")
	runningRaw := strings.TrimSpace(value(iface, "running"))
	if runningRaw == "" {
		return !disabled, nil
	}

	return !disabled && parseBool(runningRaw), nil
}

func (b *routerOSBackend) Stats(ctx context.Context, name string) (*driver.InterfaceStats, error) {
	interfaces, err := b.listEntries(ctx, "interface")
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if !strings.EqualFold(value(iface, "name"), name) {
			continue
		}

		return &driver.InterfaceStats{
			RxBytes: uint64Value(iface, "rx-byte", "rx-bytes"),
			TxBytes: uint64Value(iface, "tx-byte", "tx-bytes"),
		}, nil
	}

	return nil, nil
}

func (b *routerOSBackend) PeerStats(ctx context.Context, name string, peerPublicKey string) (*driver.PeerStats, error) {
	iface, err := b.findWireguardInterfaceByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if iface == nil {
		return nil, nil
	}

	peerEntries, err := b.interfacePeerEntries(ctx, iface)
	if err != nil {
		return nil, err
	}

	for _, p := range peerEntries {
		if !strings.EqualFold(value(p, "public-key"), peerPublicKey) {
			continue
		}

		stats := peerStatsFromEntry(p)
		return &stats, nil
	}

	return nil, nil
}

func (b *routerOSBackend) FindForeignServers(ctx context.Context, knownInterfaces []string) ([]*driver.ForeignServer, error) {
	known := make(map[string]struct{}, len(knownInterfaces))
	for _, name := range knownInterfaces {
		known[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}

	interfaces, err := b.listEntries(ctx, "interface/wireguard")
	if err != nil {
		return nil, err
	}

	foreignServers := make([]*driver.ForeignServer, 0, len(interfaces))
	for _, iface := range interfaces {
		name := strings.TrimSpace(value(iface, "name"))
		if name == "" {
			continue
		}
		if _, ok := known[strings.ToLower(name)]; ok {
			continue
		}

		peerEntries, err := b.interfacePeerEntries(ctx, iface)
		if err != nil {
			return nil, err
		}
		if shouldSkipForeignInterface(iface, peerEntries) {
			continue
		}

		addresses, err := b.interfaceAddresses(ctx, name)
		if err != nil {
			return nil, err
		}

		peers := peerEntriesToPeers(peerEntries)

		foreignServers = append(foreignServers, &driver.ForeignServer{
			Interface: &driver.ForeignInterface{
				Name:      name,
				Addresses: addresses,
				Mtu:       intValue(iface, "mtu"),
				State:     interfaceState(iface),
			},
			Name:         name,
			Description:  value(iface, "comment"),
			Type:         "wireguard",
			PublicKey:    value(iface, "public-key"),
			ListenPort:   intValue(iface, "listen-port"),
			FirewallMark: intValue(iface, "firewall-mark", "fwmark"),
			Peers:        peers,
		})
	}

	sort.Slice(foreignServers, func(i, j int) bool {
		return strings.ToLower(foreignServers[i].Name) < strings.ToLower(foreignServers[j].Name)
	})

	return foreignServers, nil
}

func (b *routerOSBackend) Close(_ context.Context) error {
	return nil
}

func (b *routerOSBackend) reconcileAddresses(ctx context.Context, interfaceName string, desiredAddress string) error {
	desiredAddress = strings.TrimSpace(desiredAddress)
	if desiredAddress == "" {
		return errors.New("interface address is required")
	}

	addressEntries, err := b.interfaceAddressEntries(ctx, interfaceName)
	if err != nil {
		return err
	}

	seenDesired := false
	for _, addressEntry := range addressEntries {
		currentAddress := strings.TrimSpace(value(addressEntry, "address"))
		if strings.EqualFold(currentAddress, desiredAddress) {
			seenDesired = true
			continue
		}

		// Keep dynamic addresses untouched.
		if boolValue(addressEntry, "dynamic") {
			continue
		}

		if err := b.deleteEntry(ctx, "ip/address", value(addressEntry, ".id")); err != nil {
			return err
		}
	}

	if !seenDesired {
		if err := b.putEntry(ctx, "ip/address", map[string]string{
			"address":   desiredAddress,
			"interface": interfaceName,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (b *routerOSBackend) reconcilePeers(ctx context.Context, iface entry, desiredPeers []*driver.PeerOptions) error {
	existingPeers, err := b.interfacePeerEntries(ctx, iface)
	if err != nil {
		return err
	}

	desiredByPublicKey := make(map[string]*driver.PeerOptions, len(desiredPeers))
	for _, p := range desiredPeers {
		if p == nil {
			continue
		}
		desiredByPublicKey[strings.TrimSpace(p.PublicKey)] = p
	}

	existingByPublicKey := make(map[string][]entry, len(existingPeers))
	dynamicByPublicKey := make(map[string]struct{}, len(existingPeers))
	for _, p := range existingPeers {
		publicKey := strings.TrimSpace(value(p, "public-key"))
		if publicKey == "" {
			continue
		}

		if boolValue(p, "dynamic") {
			dynamicByPublicKey[publicKey] = struct{}{}
			continue
		}

		existingByPublicKey[publicKey] = append(existingByPublicKey[publicKey], p)
	}

	for publicKey, peerOptions := range desiredByPublicKey {
		payload, err := peerPayload(value(iface, "name"), peerOptions)
		if err != nil {
			return err
		}

		existing := existingByPublicKey[publicKey]
		if len(existing) == 0 {
			if _, isDynamic := dynamicByPublicKey[publicKey]; isDynamic {
				continue
			}

			if err := b.putEntry(ctx, "interface/wireguard/peers", payload); err != nil {
				return err
			}
			continue
		}

		if peerNeedsPatch(existing[0], payload) {
			if err := b.patchEntry(ctx, "interface/wireguard/peers", value(existing[0], ".id"), payload); err != nil {
				return err
			}
		}

		for _, duplicate := range existing[1:] {
			if err := b.deleteEntry(ctx, "interface/wireguard/peers", value(duplicate, ".id")); err != nil {
				return err
			}
		}
	}

	for publicKey, existing := range existingByPublicKey {
		if _, ok := desiredByPublicKey[publicKey]; ok {
			continue
		}

		for _, p := range existing {
			if err := b.deleteEntry(ctx, "interface/wireguard/peers", value(p, ".id")); err != nil {
				return err
			}
		}
	}

	return nil
}

func peerPayload(interfaceName string, p *driver.PeerOptions) (map[string]string, error) {
	allowedAddress := strings.Join(p.AllowedIPs, ",")
	endpointAddress, endpointPort, err := splitEndpoint(p.Endpoint)
	if err != nil {
		return nil, err
	}

	payload := map[string]string{
		"interface":            interfaceName,
		"public-key":           p.PublicKey,
		"allowed-address":      allowedAddress,
		"endpoint-address":     endpointAddress,
		"endpoint-port":        strconv.Itoa(endpointPort),
		"persistent-keepalive": strconv.Itoa(max(0, p.PersistentKeepalive)),
		"preshared-key":        p.PresharedKey,
		"disabled":             "false",
	}

	return payload, nil
}

func splitEndpoint(endpoint string) (string, int, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", 0, nil
	}

	host, port, err := net.SplitHostPort(endpoint)
	if err == nil {
		parsedPort, parseErr := strconv.Atoi(port)
		if parseErr != nil {
			return "", 0, fmt.Errorf("invalid endpoint port %q: %w", port, parseErr)
		}
		return strings.Trim(host, "[]"), parsedPort, nil
	}

	lastColon := strings.LastIndex(endpoint, ":")
	if lastColon > 0 && lastColon < len(endpoint)-1 {
		maybePort := endpoint[lastColon+1:]
		parsedPort, parseErr := strconv.Atoi(maybePort)
		if parseErr == nil {
			hostPart := strings.Trim(endpoint[:lastColon], "[]")
			return hostPart, parsedPort, nil
		}
	}

	return strings.Trim(endpoint, "[]"), 0, nil
}

func interfaceState(iface entry) string {
	if boolValue(iface, "running") && !boolValue(iface, "disabled") {
		return "up"
	}
	return "down"
}

func (b *routerOSBackend) interfaceAddresses(ctx context.Context, interfaceName string) ([]string, error) {
	addressEntries, err := b.interfaceAddressEntries(ctx, interfaceName)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, 0, len(addressEntries))
	for _, addressEntry := range addressEntries {
		address := strings.TrimSpace(value(addressEntry, "address"))
		if address == "" {
			continue
		}
		addresses = append(addresses, address)
	}

	sort.Strings(addresses)
	return addresses, nil
}

func (b *routerOSBackend) interfaceAddressEntries(ctx context.Context, interfaceName string) ([]entry, error) {
	allAddresses, err := b.listEntries(ctx, "ip/address")
	if err != nil {
		return nil, err
	}

	filtered := make([]entry, 0, len(allAddresses))
	for _, address := range allAddresses {
		if strings.EqualFold(value(address, "interface"), interfaceName) {
			filtered = append(filtered, address)
		}
	}

	return filtered, nil
}

func (b *routerOSBackend) interfacePeers(ctx context.Context, iface entry) ([]*driver.Peer, error) {
	peerEntries, err := b.interfacePeerEntries(ctx, iface)
	if err != nil {
		return nil, err
	}

	return peerEntriesToPeers(peerEntries), nil
}

func (b *routerOSBackend) interfacePeerEntries(ctx context.Context, iface entry) ([]entry, error) {
	allPeers, err := b.listEntries(ctx, "interface/wireguard/peers")
	if err != nil {
		return nil, err
	}

	interfaceName := value(iface, "name")
	interfaceID := value(iface, ".id")

	filtered := make([]entry, 0, len(allPeers))
	for _, peerEntry := range allPeers {
		interfaceValue := value(peerEntry, "interface")
		if strings.EqualFold(interfaceValue, interfaceName) || (interfaceID != "" && interfaceValue == interfaceID) {
			filtered = append(filtered, peerEntry)
		}
	}

	return filtered, nil
}

func peerEntriesToPeers(peerEntries []entry) []*driver.Peer {
	peers := make([]*driver.Peer, 0, len(peerEntries))
	for _, peerEntry := range peerEntries {
		allowedIPs := parseAllowedIPs(value(peerEntry, "allowed-address"))
		keepalive := time.Duration(intValue(peerEntry, "persistent-keepalive")) * time.Second

		peers = append(peers, &driver.Peer{
			Name:                strings.TrimSpace(value(peerEntry, "name")),
			Description:         strings.TrimSpace(value(peerEntry, "comment")),
			PublicKey:           value(peerEntry, "public-key"),
			Endpoint:            endpointValue(peerEntry),
			AllowedIPs:          allowedIPs,
			PresharedKey:        value(peerEntry, "preshared-key"),
			PersistentKeepalive: keepalive,
			Stats:               peerStatsFromEntry(peerEntry),
		})
	}

	sort.Slice(peers, func(i, j int) bool {
		return strings.ToLower(peers[i].PublicKey) < strings.ToLower(peers[j].PublicKey)
	})
	return peers
}

func shouldSkipForeignInterface(iface entry, peerEntries []entry) bool {
	if boolValue(iface, "dynamic") {
		return true
	}

	if containsBackToHomeMarker(value(iface, "name")) || containsBackToHomeMarker(value(iface, "comment")) {
		return true
	}

	for _, peerEntry := range peerEntries {
		if boolValue(peerEntry, "dynamic") {
			return true
		}
	}

	return false
}

func containsBackToHomeMarker(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}

	return strings.Contains(normalized, "back-to-home") || strings.Contains(normalized, "back to home")
}

func interfaceNeedsPatch(existing entry, desired map[string]string) bool {
	if desired == nil {
		return false
	}

	for key, desiredValue := range desired {
		switch key {
		case "name":
			if !strings.EqualFold(strings.TrimSpace(value(existing, "name")), strings.TrimSpace(desiredValue)) {
				return true
			}
		case "private-key":
			if strings.TrimSpace(value(existing, "private-key")) != strings.TrimSpace(desiredValue) {
				return true
			}
		case "comment":
			if strings.TrimSpace(value(existing, "comment")) != strings.TrimSpace(desiredValue) {
				return true
			}
		case "mtu":
			if normalizeOptionalInt(value(existing, "mtu")) != normalizeOptionalInt(desiredValue) {
				return true
			}
		case "listen-port":
			if normalizeOptionalInt(value(existing, "listen-port")) != normalizeOptionalInt(desiredValue) {
				return true
			}
		case "disabled":
			if normalizeBoolString(value(existing, "disabled")) != normalizeBoolString(desiredValue) {
				return true
			}
		default:
			if strings.TrimSpace(value(existing, key)) != strings.TrimSpace(desiredValue) {
				return true
			}
		}
	}

	return false
}

func peerNeedsPatch(existing entry, desired map[string]string) bool {
	if desired == nil {
		return false
	}

	if !strings.EqualFold(strings.TrimSpace(value(existing, "public-key")), strings.TrimSpace(desired["public-key"])) {
		return true
	}

	if normalizeCSV(value(existing, "allowed-address")) != normalizeCSV(desired["allowed-address"]) {
		return true
	}

	if normalizeEndpointAddress(value(existing, "endpoint-address")) != normalizeEndpointAddress(desired["endpoint-address"]) {
		return true
	}

	if normalizeOptionalInt(value(existing, "endpoint-port")) != normalizeOptionalInt(desired["endpoint-port"]) {
		return true
	}

	if normalizeOptionalInt(value(existing, "persistent-keepalive")) != normalizeOptionalInt(desired["persistent-keepalive"]) {
		return true
	}

	if strings.TrimSpace(value(existing, "preshared-key")) != strings.TrimSpace(desired["preshared-key"]) {
		return true
	}

	if normalizeBoolString(value(existing, "disabled")) != normalizeBoolString(desired["disabled"]) {
		return true
	}

	return false
}

func normalizeCSV(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}

	sort.Strings(values)
	return strings.Join(values, ",")
}

func normalizeEndpointAddress(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), "[]")
}

func normalizeOptionalInt(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return raw
	}
	if parsed <= 0 {
		return ""
	}
	return strconv.Itoa(parsed)
}

func normalizeBoolString(raw string) string {
	if parseBool(raw) {
		return "true"
	}
	return "false"
}

func endpointValue(peerEntry entry) string {
	address := strings.TrimSpace(value(peerEntry, "endpoint-address"))
	if address == "" {
		return ""
	}

	port := intValue(peerEntry, "endpoint-port")
	if port <= 0 {
		return address
	}

	if ip, err := netip.ParseAddr(address); err == nil && ip.Is6() {
		return "[" + address + "]:" + strconv.Itoa(port)
	}

	return address + ":" + strconv.Itoa(port)
}

func peerStatsFromEntry(peerEntry entry) driver.PeerStats {
	lastHandshake := time.Time{}
	if seconds := intValue(peerEntry, "last-handshake", "last-handshake-time"); seconds > 0 {
		lastHandshake = time.Now().Add(-time.Duration(seconds) * time.Second)
	}

	return driver.PeerStats{
		LastHandshakeTime: lastHandshake,
		ReceiveBytes:      int64(uint64Value(peerEntry, "rx", "rx-byte", "rx-bytes")),
		TransmitBytes:     int64(uint64Value(peerEntry, "tx", "tx-byte", "tx-bytes")),
		ProtocolVersion:   intValue(peerEntry, "protocol-version"),
	}
}

func parseAllowedIPs(allowed string) []net.IPNet {
	if strings.TrimSpace(allowed) == "" {
		return nil
	}

	parts := strings.Split(allowed, ",")
	allowedIPs := make([]net.IPNet, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		_, cidr, err := net.ParseCIDR(part)
		if err != nil {
			continue
		}
		allowedIPs = append(allowedIPs, *cidr)
	}

	return allowedIPs
}

func (b *routerOSBackend) findWireguardInterfaceByName(ctx context.Context, name string) (entry, error) {
	interfaces, err := b.listEntries(ctx, "interface/wireguard")
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if strings.EqualFold(value(iface, "name"), name) {
			return iface, nil
		}
	}

	return nil, nil
}

func (b *routerOSBackend) putEntry(ctx context.Context, resource string, payload map[string]string) error {
	_, err := b.request(ctx, http.MethodPut, resource, payload)
	return err
}

func (b *routerOSBackend) patchEntry(ctx context.Context, resource string, id string, payload map[string]string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("routeros %s patch requires .id", resource)
	}

	_, err := b.request(ctx, http.MethodPatch, resource+"/"+id, payload)
	if errors.Is(err, errRouterOSResourceNotFound) {
		return nil
	}
	return err
}

func (b *routerOSBackend) deleteEntry(ctx context.Context, resource string, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	_, err := b.request(ctx, http.MethodDelete, resource+"/"+id, nil)
	if errors.Is(err, errRouterOSResourceNotFound) {
		return nil
	}
	return err
}

func (b *routerOSBackend) listEntries(ctx context.Context, resource string) ([]entry, error) {
	responseBytes, err := b.request(ctx, http.MethodGet, resource, nil)
	if err != nil {
		if errors.Is(err, errRouterOSResourceNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return decodeEntries(responseBytes)
}

func (b *routerOSBackend) request(ctx context.Context, method string, resource string, payload map[string]string) ([]byte, error) {
	urlPath := strings.TrimPrefix(strings.TrimSpace(resource), "/")
	requestURL := strings.TrimRight(b.baseURL, "/") + "/" + urlPath

	var requestBody io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(bodyBytes)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(b.username, b.password)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := b.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("routeros api request failed: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read routeros api response: %w", err)
	}

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return responseBody, nil
	}

	message := strings.TrimSpace(string(responseBody))
	if message == "" {
		message = response.Status
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", errRouterOSResourceNotFound, message)
	}

	return nil, fmt.Errorf("routeros api %s %s failed: %s", method, resource, message)
}

func decodeEntries(responseBytes []byte) ([]entry, error) {
	trimmed := strings.TrimSpace(string(responseBytes))
	if trimmed == "" {
		return nil, nil
	}

	switch trimmed[0] {
	case '[':
		var raw []map[string]any
		if err := json.Unmarshal(responseBytes, &raw); err != nil {
			return nil, err
		}

		entries := make([]entry, 0, len(raw))
		for _, item := range raw {
			entries = append(entries, normalizeEntry(item))
		}
		return entries, nil
	case '{':
		var raw map[string]any
		if err := json.Unmarshal(responseBytes, &raw); err != nil {
			return nil, err
		}
		return []entry{normalizeEntry(raw)}, nil
	default:
		return nil, fmt.Errorf("unexpected routeros api response: %s", trimmed)
	}
}

func normalizeEntry(raw map[string]any) entry {
	normalized := make(entry, len(raw))
	for key, value := range raw {
		normalized[key] = stringify(value)
	}
	return normalized
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		if math.Mod(typed, 1) == 0 {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}

func value(values entry, keys ...string) string {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			return strings.TrimSpace(raw)
		}
	}
	return ""
}

func intValue(values entry, keys ...string) int {
	raw := value(values, keys...)
	if raw == "" {
		return 0
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return parsed
}

func uint64Value(values entry, keys ...string) uint64 {
	raw := value(values, keys...)
	if raw == "" {
		return 0
	}

	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func boolValue(values entry, keys ...string) bool {
	return parseBool(value(values, keys...))
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "yes", "on", "1":
		return true
	default:
		return false
	}
}

func parseURL(rawURL string) (*parsedURL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid routeros backend url: %w", err)
	}

	if !strings.EqualFold(parsed.Scheme, "routeros") {
		return nil, fmt.Errorf("invalid backend type: %s", parsed.Scheme)
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nil, errors.New("routeros backend requires host")
	}

	if parsed.User == nil {
		return nil, errors.New("routeros backend requires user and password")
	}
	username := strings.TrimSpace(parsed.User.Username())
	password, hasPassword := parsed.User.Password()
	if username == "" || !hasPassword || password == "" {
		return nil, errors.New("routeros backend requires user and password")
	}

	useHTTPS, err := queryBool(parsed.Query(), "https", true)
	if err != nil {
		return nil, err
	}
	if !useHTTPS {
		return nil, errors.New("routeros backend supports HTTPS only")
	}
	insecureSkipVerify, err := queryBool(parsed.Query(), "insecureSkipVerify", false)
	if err != nil {
		return nil, err
	}

	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		port = defaultHTTPSPort
	}

	path := strings.TrimSpace(parsed.Path)
	if path == "" || path == "/" {
		path = defaultRESTPath
	} else {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		path = strings.TrimRight(path, "/")
		if path == "" {
			path = defaultRESTPath
		}
	}

	baseURL := fmt.Sprintf("https://%s%s", net.JoinHostPort(host, port), path)
	return &parsedURL{
		baseURL:            baseURL,
		username:           username,
		password:           password,
		insecureSkipVerify: insecureSkipVerify,
	}, nil
}

func queryBool(values url.Values, key string, defaultValue bool) (bool, error) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return defaultValue, nil
	}

	if parsed, err := strconv.ParseBool(raw); err == nil {
		return parsed, nil
	}

	switch strings.ToLower(raw) {
	case "yes", "on", "1":
		return true, nil
	case "no", "off", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s value %q", key, raw)
	}
}
