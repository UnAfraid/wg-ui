//go:build darwin

package exec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/bits"
	"net"
	"net/netip"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	backendpkg "github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

func init() {
	backend.Register("exec", NewExecBackend, isExecBackendAvailable())
}

func isExecBackendAvailable() bool {
	for _, cmd := range []string{"wg", "wg-quick", "ifconfig", "netstat", "route"} {
		if _, err := osexec.LookPath(cmd); err != nil {
			return false
		}
	}
	return true
}

type execBackend struct {
	configDir    string
	useSudo      bool
	sudoPath     string
	wgPath       string
	wgQuickPath  string
	ifconfigPath string
	netstatPath  string
	routePath    string
	installPath  string
	findPath     string
	catPath      string
}

func NewExecBackend(rawURL string) (backend.Backend, error) {
	parsed, err := backendpkg.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse backend url: %w", err)
	}

	if parsed.Type != "exec" {
		return nil, fmt.Errorf("invalid backend type: %s", parsed.Type)
	}

	if parsed.Host != "" || parsed.Port != "" || parsed.User != "" {
		return nil, errors.New("exec backend does not support remote host/user/port in url")
	}

	configDir := strings.TrimSpace(parsed.Path)
	if configDir == "" {
		configDir = defaultConfigDir
	}
	if !filepath.IsAbs(configDir) {
		return nil, fmt.Errorf("exec backend path must be absolute: %q", configDir)
	}
	configDir = filepath.Clean(configDir)

	useSudo := false
	if rawUseSudo, ok := parsed.Options["sudo"]; ok && strings.TrimSpace(rawUseSudo) != "" {
		parsedUseSudo, err := strconv.ParseBool(rawUseSudo)
		if err != nil {
			return nil, fmt.Errorf("invalid sudo option %q: %w", rawUseSudo, err)
		}
		useSudo = parsedUseSudo
	}

	wgPath, err := osexec.LookPath("wg")
	if err != nil {
		return nil, fmt.Errorf("failed to find wg command: %w", err)
	}
	wgQuickPath, err := osexec.LookPath("wg-quick")
	if err != nil {
		return nil, fmt.Errorf("failed to find wg-quick command: %w", err)
	}
	ifconfigPath, err := osexec.LookPath("ifconfig")
	if err != nil {
		return nil, fmt.Errorf("failed to find ifconfig command: %w", err)
	}
	netstatPath, err := osexec.LookPath("netstat")
	if err != nil {
		return nil, fmt.Errorf("failed to find netstat command: %w", err)
	}
	routePath, err := osexec.LookPath("route")
	if err != nil {
		return nil, fmt.Errorf("failed to find route command: %w", err)
	}
	installPath, err := osexec.LookPath("install")
	if err != nil {
		return nil, fmt.Errorf("failed to find install command: %w", err)
	}
	findPath, err := osexec.LookPath("find")
	if err != nil {
		return nil, fmt.Errorf("failed to find find command: %w", err)
	}
	catPath, err := osexec.LookPath("cat")
	if err != nil {
		return nil, fmt.Errorf("failed to find cat command: %w", err)
	}

	sudoPath := ""
	if useSudo {
		sudoPath, err = osexec.LookPath("sudo")
		if err != nil {
			return nil, fmt.Errorf("failed to find sudo command: %w", err)
		}
	}

	return &execBackend{
		configDir:    configDir,
		useSudo:      useSudo,
		sudoPath:     sudoPath,
		wgPath:       wgPath,
		wgQuickPath:  wgQuickPath,
		ifconfigPath: ifconfigPath,
		netstatPath:  netstatPath,
		routePath:    routePath,
		installPath:  installPath,
		findPath:     findPath,
		catPath:      catPath,
	}, nil
}

func (b *execBackend) Device(ctx context.Context, name string) (*backend.Device, error) {
	device, err := b.deviceFromDump(ctx, name)
	if err != nil {
		exists, existsErr := b.interfaceExists(ctx, name)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return nil, err
		}
		return b.deviceFromConfigFile(ctx, name)
	}

	details, err := b.getInterfaceDetails(ctx, name)
	if err != nil {
		return nil, err
	}

	device.Interface = backend.Interface{
		Name:      name,
		Addresses: details.Addresses,
		Mtu:       details.MTU,
	}
	return device, nil
}

func (b *execBackend) Up(ctx context.Context, options backend.ConfigureOptions) (*backend.Device, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	name := options.InterfaceOptions.Name
	configPath, err := b.writeConfig(ctx, options)
	if err != nil {
		return nil, err
	}

	exists, err := b.interfaceExists(ctx, name)
	if err != nil {
		return nil, err
	}

	if !exists {
		if _, err := b.runWGQuick(ctx, "up", configPath); err != nil {
			return nil, fmt.Errorf("failed to bring interface up: %w", err)
		}
	} else {
		if err := b.syncWithoutRestart(ctx, options, configPath); err != nil {
			logrus.WithError(err).
				WithField("interface", name).
				Warn("live reconfiguration failed, restarting interface")

			if downErr := b.down(ctx, name, configPath); downErr != nil {
				logrus.WithError(downErr).
					WithField("interface", name).
					Warn("failed to stop interface during reconfiguration fallback")
			}

			if _, err := b.runWGQuick(ctx, "up", configPath); err != nil {
				return nil, fmt.Errorf("failed to restart interface after live reconfiguration failure: %w", err)
			}
		}
	}

	return b.Device(ctx, name)
}

func (b *execBackend) Down(ctx context.Context, name string) error {
	return b.down(ctx, name, b.configFilePath(name))
}

func (b *execBackend) down(ctx context.Context, name string, configPath string) error {
	exists, err := b.interfaceExists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if _, err := b.runWGQuick(ctx, "down", configPath); err != nil {
		if _, fallbackErr := b.runWGQuick(ctx, "down", name); fallbackErr != nil {
			return err
		}
	}
	return nil
}

func (b *execBackend) Status(ctx context.Context, name string) (bool, error) {
	return b.interfaceExists(ctx, name)
}

func (b *execBackend) Stats(ctx context.Context, name string) (*backend.InterfaceStats, error) {
	stats, err := b.interfaceStats(ctx, name)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return &backend.InterfaceStats{}, nil
	}
	return stats, nil
}

func (b *execBackend) PeerStats(ctx context.Context, name string, peerPublicKey string) (*backend.PeerStats, error) {
	device, err := b.deviceFromDump(ctx, name)
	if err != nil {
		return nil, err
	}

	for _, p := range device.Wireguard.Peers {
		if p.PublicKey == peerPublicKey {
			return &p.Stats, nil
		}
	}
	return nil, nil
}

func (b *execBackend) FindForeignServers(ctx context.Context, knownInterfaces []string) ([]*backend.ForeignServer, error) {
	knownInterfaceSet := make(map[string]struct{}, len(knownInterfaces))
	for _, name := range knownInterfaces {
		knownInterfaceSet[name] = struct{}{}
	}

	existingForeignNames := make(map[string]struct{})
	var foreignServers []*backend.ForeignServer

	interfacesOutput, err := b.runWG(ctx, "show", "interfaces")
	if err != nil {
		return nil, fmt.Errorf("failed to list wireguard interfaces: %w", err)
	}

	for _, name := range strings.Fields(strings.TrimSpace(string(interfacesOutput))) {
		if _, ok := knownInterfaceSet[name]; ok {
			continue
		}

		device, err := b.deviceFromDump(ctx, name)
		if err != nil {
			return nil, err
		}

		details, detailsErr := b.getInterfaceDetails(ctx, name)
		foreignInterface := &backend.ForeignInterface{
			Name:  name,
			State: "up",
		}
		if detailsErr == nil {
			foreignInterface.Addresses = details.Addresses
			foreignInterface.Mtu = details.MTU
			if details.State != "" {
				foreignInterface.State = details.State
			}
		}

		foreignServers = append(foreignServers, &backend.ForeignServer{
			Interface:    foreignInterface,
			Name:         device.Wireguard.Name,
			Type:         "wireguard",
			PublicKey:    device.Wireguard.PublicKey,
			ListenPort:   device.Wireguard.ListenPort,
			FirewallMark: device.Wireguard.FirewallMark,
			Peers:        device.Wireguard.Peers,
		})
		existingForeignNames[name] = struct{}{}
	}

	fileForeignServers, err := b.findForeignServersFromConfigFiles(ctx, knownInterfaceSet, existingForeignNames)
	if err != nil {
		return nil, err
	}
	foreignServers = append(foreignServers, fileForeignServers...)
	return foreignServers, nil
}

func (b *execBackend) Close(_ context.Context) error {
	return nil
}

func (b *execBackend) Supported() bool {
	return true
}

func (b *execBackend) syncWithoutRestart(ctx context.Context, options backend.ConfigureOptions, configPath string) error {
	name := options.InterfaceOptions.Name

	strippedConfig, err := b.runWGQuick(ctx, "strip", configPath)
	if err != nil {
		return fmt.Errorf("failed to strip config for syncconf: %w", err)
	}

	tmpPath, err := writeTempFile(strippedConfig)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if _, err := b.runWG(ctx, "syncconf", name, tmpPath); err != nil {
		return fmt.Errorf("failed to apply syncconf: %w", err)
	}

	if err := b.ensureInterfaceSettings(ctx, name, options.InterfaceOptions.Address, options.InterfaceOptions.Mtu); err != nil {
		return err
	}

	if err := b.reconcileRoutes(ctx, name, options.InterfaceOptions.Address, options.WireguardOptions.Peers); err != nil {
		return err
	}

	return nil
}

func (b *execBackend) ensureInterfaceSettings(ctx context.Context, name string, desiredAddress string, desiredMTU int) error {
	details, err := b.getInterfaceDetails(ctx, name)
	if err != nil {
		return err
	}

	desired, err := normalizeCIDR(desiredAddress)
	if err != nil {
		return fmt.Errorf("failed to parse desired address %q: %w", desiredAddress, err)
	}

	foundAddress := false
	for _, current := range details.Addresses {
		normalizedCurrent, err := normalizeCIDR(current)
		if err != nil {
			continue
		}
		if normalizedCurrent == desired {
			foundAddress = true
			break
		}
	}
	if !foundAddress {
		return fmt.Errorf("interface address differs from desired address %s", desired)
	}

	if desiredMTU > 0 && details.MTU != desiredMTU {
		if _, err := b.runIfconfig(ctx, name, "mtu", strconv.Itoa(desiredMTU)); err != nil {
			return fmt.Errorf("failed to set mtu %d for %s: %w", desiredMTU, name, err)
		}
	}

	return nil
}

func (b *execBackend) reconcileRoutes(ctx context.Context, name string, interfaceAddress string, peers []*backend.PeerOptions) error {
	desiredIPv4, desiredIPv6, err := collectDesiredRoutes(peers)
	if err != nil {
		return err
	}

	preserveIPv4 := make(map[string]struct{})
	preserveIPv6 := make(map[string]struct{})
	if interfacePrefix, err := normalizeCIDR(interfaceAddress); err == nil {
		parsedPrefix, parseErr := netip.ParsePrefix(interfacePrefix)
		if parseErr == nil {
			if parsedPrefix.Addr().Is4() {
				preserveIPv4[interfacePrefix] = struct{}{}
			} else {
				preserveIPv6[interfacePrefix] = struct{}{}
			}
		}
	}

	currentIPv4, err := b.listRoutes(ctx, name, false)
	if err != nil {
		return err
	}
	currentIPv6, err := b.listRoutes(ctx, name, true)
	if err != nil {
		return err
	}

	for prefix := range desiredIPv4 {
		if _, ok := currentIPv4[prefix]; ok {
			continue
		}
		if err := b.ensureRoute(ctx, name, prefix, false); err != nil {
			return err
		}
	}

	for prefix := range desiredIPv6 {
		if _, ok := currentIPv6[prefix]; ok {
			continue
		}
		if err := b.ensureRoute(ctx, name, prefix, true); err != nil {
			return err
		}
	}

	for prefix := range currentIPv4 {
		if _, keep := desiredIPv4[prefix]; keep {
			continue
		}
		if _, preserve := preserveIPv4[prefix]; preserve {
			continue
		}
		if err := b.deleteRoute(ctx, name, prefix, false); err != nil {
			return err
		}
	}

	for prefix := range currentIPv6 {
		if _, keep := desiredIPv6[prefix]; keep {
			continue
		}
		if _, preserve := preserveIPv6[prefix]; preserve {
			continue
		}
		if err := b.deleteRoute(ctx, name, prefix, true); err != nil {
			return err
		}
	}

	return nil
}

func collectDesiredRoutes(peers []*backend.PeerOptions) (map[string]struct{}, map[string]struct{}, error) {
	desiredIPv4 := make(map[string]struct{})
	desiredIPv6 := make(map[string]struct{})

	for _, p := range peers {
		for _, allowedIP := range p.AllowedIPs {
			prefix, err := normalizeCIDR(allowedIP)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse allowed ip %q: %w", allowedIP, err)
			}

			parsed, err := netip.ParsePrefix(prefix)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse allowed ip %q: %w", prefix, err)
			}
			if parsed.Addr().Is4() {
				desiredIPv4[prefix] = struct{}{}
			} else {
				desiredIPv6[prefix] = struct{}{}
			}
		}
	}

	return desiredIPv4, desiredIPv6, nil
}

func (b *execBackend) ensureRoute(ctx context.Context, name string, prefix string, ipv6 bool) error {
	family := "-inet"
	if ipv6 {
		family = "-inet6"
	}

	_, err := b.runRoute(ctx, "-n", "-q", "add", family, "-net", prefix, "-interface", name)
	if err == nil {
		return nil
	}

	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "exists") {
		return fmt.Errorf("failed to add route %s via %s: %w", prefix, name, err)
	}

	if _, err := b.runRoute(ctx, "-n", "-q", "change", family, "-net", prefix, "-interface", name); err != nil {
		return fmt.Errorf("failed to update route %s via %s: %w", prefix, name, err)
	}
	return nil
}

func (b *execBackend) deleteRoute(ctx context.Context, name string, prefix string, ipv6 bool) error {
	family := "-inet"
	if ipv6 {
		family = "-inet6"
	}

	_, err := b.runRoute(ctx, "-n", "-q", "delete", family, "-net", prefix, "-interface", name)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "not in table") || strings.Contains(lower, "no such process") {
			return nil
		}
		return fmt.Errorf("failed to delete route %s via %s: %w", prefix, name, err)
	}
	return nil
}

func (b *execBackend) listRoutes(ctx context.Context, name string, ipv6 bool) (map[string]struct{}, error) {
	family := "inet"
	defaultPrefix := "0.0.0.0/0"
	if ipv6 {
		family = "inet6"
		defaultPrefix = "::/0"
	}

	output, err := b.runNetstat(ctx, "-rn", "-f", family)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	result := make(map[string]struct{})
	lines := strings.Split(string(output), "\n")

	destinationIndex := -1
	netifIndex := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}

		if destinationIndex == -1 || netifIndex == -1 {
			for idx, field := range fields {
				if field == "Destination" {
					destinationIndex = idx
				}
				if field == "Netif" {
					netifIndex = idx
				}
			}
			continue
		}

		if strings.HasPrefix(trimmed, "Internet") {
			continue
		}

		if len(fields) <= netifIndex || len(fields) <= destinationIndex {
			continue
		}

		if fields[netifIndex] != name {
			continue
		}

		dst := fields[destinationIndex]
		if dst == "default" {
			dst = defaultPrefix
		}

		prefix, normalizeErr := normalizeRoutePrefix(dst, ipv6)
		if normalizeErr != nil {
			continue
		}
		result[prefix] = struct{}{}
	}

	return result, nil
}

func normalizeRoutePrefix(value string, ipv6 bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("empty route destination")
	}

	if strings.Contains(value, "/") {
		return normalizeCIDR(value)
	}

	if strings.Contains(value, "%") {
		value = strings.Split(value, "%")[0]
	}

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return "", err
	}

	prefixLen := 32
	if ipv6 {
		prefixLen = 128
	}
	return netip.PrefixFrom(addr, prefixLen).Masked().String(), nil
}

func (b *execBackend) interfaceExists(ctx context.Context, name string) (bool, error) {
	_, err := b.runIfconfig(ctx, name)
	if err == nil {
		return true, nil
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "does not exist") || strings.Contains(lower, "no such interface") {
		return false, nil
	}
	return false, err
}

func (b *execBackend) interfaceStats(ctx context.Context, name string) (*backend.InterfaceStats, error) {
	output, err := b.runNetstat(ctx, "-bI", name)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "does not exist") || strings.Contains(lower, "no such interface") {
			return nil, nil
		}
		return nil, err
	}

	return parseNetstatInterfaceStats(name, string(output)), nil
}

func parseNetstatInterfaceStats(name string, output string) *backend.InterfaceStats {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return &backend.InterfaceStats{}
	}

	headerIndex := -1
	var headerFields []string
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) > 0 && fields[0] == "Name" {
			headerIndex = idx
			headerFields = fields
			break
		}
	}
	if headerIndex == -1 {
		return &backend.InterfaceStats{}
	}

	headerMap := make(map[string]int, len(headerFields))
	for i, field := range headerFields {
		headerMap[field] = i
	}

	for _, line := range lines[headerIndex+1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		nameIdx, ok := headerMap["Name"]
		if !ok || len(fields) <= nameIdx {
			continue
		}
		if fields[nameIdx] != name {
			continue
		}

		return &backend.InterfaceStats{
			RxPackets: parseFieldUint64(fields, headerMap, "Ipkts"),
			TxPackets: parseFieldUint64(fields, headerMap, "Opkts"),
			RxErrors:  parseFieldUint64(fields, headerMap, "Ierrs"),
			TxErrors:  parseFieldUint64(fields, headerMap, "Oerrs"),
			Collisions: func() uint64 {
				if _, ok := headerMap["Coll"]; ok {
					return parseFieldUint64(fields, headerMap, "Coll")
				}
				return 0
			}(),
			RxDropped: parseFieldUint64(fields, headerMap, "Drop"),
			RxBytes:   parseFieldUint64(fields, headerMap, "Ibytes"),
			TxBytes:   parseFieldUint64(fields, headerMap, "Obytes"),
		}
	}

	return &backend.InterfaceStats{}
}

func parseFieldUint64(fields []string, headerMap map[string]int, key string) uint64 {
	idx, ok := headerMap[key]
	if !ok || idx >= len(fields) {
		return 0
	}
	value := strings.TrimSpace(fields[idx])
	if value == "" || value == "-" {
		return 0
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func (b *execBackend) getInterfaceDetails(ctx context.Context, name string) (*interfaceDetails, error) {
	output, err := b.runIfconfig(ctx, name)
	if err != nil {
		return nil, err
	}

	details := &interfaceDetails{State: "up"}
	addressSet := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "status:") {
			details.State = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			continue
		}

		if strings.Contains(line, " mtu ") {
			fields := strings.Fields(line)
			for i := 0; i < len(fields)-1; i++ {
				if fields[i] == "mtu" {
					mtu, parseErr := strconv.Atoi(fields[i+1])
					if parseErr == nil {
						details.MTU = mtu
					}
					break
				}
			}
		}

		if strings.HasPrefix(line, "inet6 ") {
			if addr, parseErr := parseIfconfigIPv6Address(line); parseErr == nil {
				addressSet[addr] = struct{}{}
			}
			continue
		}

		if strings.HasPrefix(line, "inet ") {
			if addr, parseErr := parseIfconfigIPv4Address(line); parseErr == nil {
				addressSet[addr] = struct{}{}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse ifconfig output for %s: %w", name, err)
	}

	for address := range addressSet {
		details.Addresses = append(details.Addresses, address)
	}
	sort.Strings(details.Addresses)

	return details, nil
}

func parseIfconfigIPv4Address(line string) (string, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", errors.New("invalid inet line")
	}

	ipAddr, err := netip.ParseAddr(fields[1])
	if err != nil {
		return "", err
	}

	prefixLen := 32
	for i := 2; i < len(fields)-1; i++ {
		if fields[i] == "netmask" {
			prefixLen, err = parseIfconfigMaskPrefix(fields[i+1])
			if err != nil {
				return "", err
			}
			break
		}
	}

	return netip.PrefixFrom(ipAddr, prefixLen).Masked().String(), nil
}

func parseIfconfigIPv6Address(line string) (string, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", errors.New("invalid inet6 line")
	}

	address := strings.Split(fields[1], "%")[0]
	ipAddr, err := netip.ParseAddr(address)
	if err != nil {
		return "", err
	}

	prefixLen := 128
	for i := 2; i < len(fields)-1; i++ {
		if fields[i] == "prefixlen" {
			prefixLen, err = strconv.Atoi(fields[i+1])
			if err != nil {
				return "", err
			}
			break
		}
	}

	return netip.PrefixFrom(ipAddr, prefixLen).Masked().String(), nil
}

func parseIfconfigMaskPrefix(maskValue string) (int, error) {
	if strings.HasPrefix(maskValue, "0x") || strings.HasPrefix(maskValue, "0X") {
		parsedMask, err := strconv.ParseUint(maskValue, 0, 32)
		if err != nil {
			return 0, err
		}
		return bits.OnesCount32(uint32(parsedMask)), nil
	}

	ip := net.ParseIP(maskValue)
	if ip == nil {
		return 0, fmt.Errorf("invalid netmask: %s", maskValue)
	}

	mask := net.IPMask(ip.To4())
	if mask == nil {
		return 0, fmt.Errorf("invalid ipv4 netmask: %s", maskValue)
	}

	ones, bitsCount := mask.Size()
	if bitsCount != 32 {
		return 0, fmt.Errorf("invalid netmask size: %d", bitsCount)
	}

	return ones, nil
}

func (b *execBackend) writeConfig(ctx context.Context, options backend.ConfigureOptions) (string, error) {
	name := options.InterfaceOptions.Name
	configPath := b.configFilePath(name)
	configContent := renderConfig(options)

	if b.useSudo {
		tmpPath, err := writeTempFile([]byte(configContent))
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpPath)

		if _, err := b.runCommand(ctx, b.installPath, "-d", "-m", "0755", b.configDir); err != nil {
			return "", fmt.Errorf("failed to prepare config directory %s: %w", b.configDir, err)
		}
		if _, err := b.runCommand(ctx, b.installPath, "-m", "0600", tmpPath, configPath); err != nil {
			return "", fmt.Errorf("failed to install config file %s: %w", configPath, err)
		}
		return configPath, nil
	}

	if err := os.MkdirAll(b.configDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", b.configDir, err)
	}

	tmpFile, err := os.CreateTemp(b.configDir, fmt.Sprintf(".%s-*.conf", name))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tmpPath := tmpFile.Name()

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmpFile.WriteString(configContent); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to write temporary config file: %w", err)
	}
	if err := tmpFile.Chmod(0o600); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to chmod temporary config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to close temporary config file: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to move config into place: %w", err)
	}

	return configPath, nil
}

func (b *execBackend) deviceFromConfigFile(ctx context.Context, name string) (*backend.Device, error) {
	configPath := b.configFilePath(name)
	configContent, err := b.readConfigFile(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file for interface %s: %w", name, err)
	}

	parsedConfig, err := parseConfigDevice(name, string(configContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file for interface %s: %w", name, err)
	}

	return parsedConfig.Device, nil
}

func (b *execBackend) findForeignServersFromConfigFiles(
	ctx context.Context,
	knownInterfaceSet map[string]struct{},
	existingForeignNames map[string]struct{},
) ([]*backend.ForeignServer, error) {
	configFiles, err := b.listConfigFiles(ctx)
	if err != nil {
		return nil, err
	}

	var foreignServers []*backend.ForeignServer
	for _, configPath := range configFiles {
		base := filepath.Base(configPath)
		name := strings.TrimSuffix(base, ".conf")
		if strings.TrimSpace(name) == "" || name == base {
			continue
		}
		if _, ok := knownInterfaceSet[name]; ok {
			continue
		}
		if _, ok := existingForeignNames[name]; ok {
			continue
		}

		configContent, err := b.readConfigFile(ctx, configPath)
		if err != nil {
			logrus.WithError(err).WithField("path", configPath).Warn("failed to read foreign server config file")
			continue
		}

		parsedConfig, err := parseConfigDevice(name, string(configContent))
		if err != nil {
			logrus.WithError(err).WithField("path", configPath).Warn("failed to parse foreign server config file")
			continue
		}

		foreignServers = append(foreignServers, &backend.ForeignServer{
			Interface: &backend.ForeignInterface{
				Name:      parsedConfig.Device.Interface.Name,
				Addresses: parsedConfig.Device.Interface.Addresses,
				Mtu:       parsedConfig.Device.Interface.Mtu,
				State:     "down",
			},
			Name:         parsedConfig.Device.Wireguard.Name,
			Description:  parsedConfig.Description,
			Type:         "wireguard",
			PublicKey:    parsedConfig.Device.Wireguard.PublicKey,
			ListenPort:   parsedConfig.Device.Wireguard.ListenPort,
			FirewallMark: parsedConfig.Device.Wireguard.FirewallMark,
			Peers:        parsedConfig.Device.Wireguard.Peers,
		})
	}

	return foreignServers, nil
}

func (b *execBackend) listConfigFiles(ctx context.Context) ([]string, error) {
	if b.useSudo {
		output, err := b.runCommand(ctx, b.findPath, b.configDir, "-maxdepth", "1", "-type", "f", "-name", "*.conf", "-print")
		if err != nil {
			lower := strings.ToLower(err.Error())
			if strings.Contains(lower, "no such file or directory") {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list config files with sudo in %s: %w", b.configDir, err)
		}

		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return nil, nil
		}

		files := strings.Split(trimmed, "\n")
		for i := range files {
			files[i] = strings.TrimSpace(files[i])
		}
		sort.Strings(files)
		return files, nil
	}

	entries, err := os.ReadDir(b.configDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list config files in %s: %w", b.configDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}
		files = append(files, filepath.Join(b.configDir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func (b *execBackend) readConfigFile(ctx context.Context, path string) ([]byte, error) {
	if b.useSudo {
		output, err := b.runCommand(ctx, b.catPath, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s with sudo: %w", path, err)
		}
		return output, nil
	}

	output, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	return output, nil
}

func (b *execBackend) deviceFromDump(ctx context.Context, name string) (*backend.Device, error) {
	output, err := b.runWG(ctx, "show", name, "dump")
	if err != nil {
		return nil, fmt.Errorf("failed to read device dump for %s: %w", name, err)
	}
	return parseDeviceDump(name, string(output))
}

func (b *execBackend) configFilePath(name string) string {
	return filepath.Join(b.configDir, fmt.Sprintf("%s.conf", name))
}

func (b *execBackend) runWG(ctx context.Context, args ...string) ([]byte, error) {
	return b.runCommand(ctx, b.wgPath, args...)
}

func (b *execBackend) runWGQuick(ctx context.Context, args ...string) ([]byte, error) {
	return b.runCommand(ctx, b.wgQuickPath, args...)
}

func (b *execBackend) runIfconfig(ctx context.Context, args ...string) ([]byte, error) {
	return b.runCommand(ctx, b.ifconfigPath, args...)
}

func (b *execBackend) runNetstat(ctx context.Context, args ...string) ([]byte, error) {
	return b.runCommand(ctx, b.netstatPath, args...)
}

func (b *execBackend) runRoute(ctx context.Context, args ...string) ([]byte, error) {
	return b.runCommand(ctx, b.routePath, args...)
}

func (b *execBackend) runCommand(ctx context.Context, binary string, args ...string) ([]byte, error) {
	cmdName := binary
	cmdArgs := args
	if b.useSudo {
		cmdName = b.sudoPath
		cmdArgs = append([]string{"-n", binary}, args...)
	}

	cmd := osexec.CommandContext(ctx, cmdName, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if b.useSudo {
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "a password is required") || strings.Contains(lower, "no tty present") {
				return nil, fmt.Errorf("passwordless sudo is required for command %s %s", filepath.Base(binary), strings.Join(args, " "))
			}
		}

		if trimmed == "" {
			return nil, fmt.Errorf("command failed: %s %s: %w", filepath.Base(binary), strings.Join(args, " "), err)
		}
		return nil, fmt.Errorf("command failed: %s %s: %w: %s", filepath.Base(binary), strings.Join(args, " "), err, trimmed)
	}

	return output, nil
}

type interfaceDetails struct {
	Addresses []string
	MTU       int
	State     string
}
