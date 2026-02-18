//go:build linux

package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func Register() {
	driver.Register("exec", func(_ context.Context, rawURL string) (driver.Backend, error) {
		return NewExecBackend(rawURL)
	}, isExecBackendAvailable())
}

func isExecBackendAvailable() bool {
	for _, cmd := range []string{"wg", "wg-quick", "ip"} {
		if _, err := osexec.LookPath(cmd); err != nil {
			return false
		}
	}
	return true
}

type execBackend struct {
	configDir   string
	useSudo     bool
	sudoPath    string
	wgPath      string
	wgQuickPath string
	ipPath      string
}

func NewExecBackend(rawURL string) (driver.Backend, error) {
	parsed, err := backend.ParseURL(rawURL)
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
	ipPath, err := osexec.LookPath("ip")
	if err != nil {
		return nil, fmt.Errorf("failed to find ip command: %w", err)
	}

	sudoPath := ""
	if useSudo {
		sudoPath, err = osexec.LookPath("sudo")
		if err != nil {
			return nil, fmt.Errorf("failed to find sudo command: %w", err)
		}
	}

	return &execBackend{
		configDir:   configDir,
		useSudo:     useSudo,
		sudoPath:    sudoPath,
		wgPath:      wgPath,
		wgQuickPath: wgQuickPath,
		ipPath:      ipPath,
	}, nil
}

func (b *execBackend) Device(ctx context.Context, name string) (*driver.Device, error) {
	device, err := b.deviceFromDump(ctx, name)
	if err != nil {
		exists, existsErr := b.interfaceExists(name)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return nil, err
		}
		return b.deviceFromConfigFile(ctx, name)
	}

	link, err := findInterface(name)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, fmt.Errorf("interface not found: %s", name)
	}

	addressList, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s address list: %w", name, err)
	}

	device.Interface = driver.Interface{
		Name: link.Attrs().Name,
		Addresses: mapArray(addressList, func(addr netlink.Addr) string {
			return addr.String()
		}),
		Mtu: link.Attrs().MTU,
	}

	return device, nil
}

func (b *execBackend) Up(ctx context.Context, options driver.ConfigureOptions) (*driver.Device, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	name := options.InterfaceOptions.Name
	configPath, err := b.writeConfig(ctx, options)
	if err != nil {
		return nil, err
	}

	exists, err := b.interfaceExists(name)
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
	configPath := b.configFilePath(name)
	return b.down(ctx, name, configPath)
}

func (b *execBackend) down(ctx context.Context, name string, configPath string) error {
	exists, err := b.interfaceExists(name)
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

func (b *execBackend) Status(_ context.Context, name string) (bool, error) {
	return b.interfaceExists(name)
}

func (b *execBackend) Stats(_ context.Context, name string) (*driver.InterfaceStats, error) {
	link, err := findInterface(name)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, nil
	}

	return linkStatisticsToBackendInterfaceStats(link.Attrs().Statistics), nil
}

func (b *execBackend) PeerStats(ctx context.Context, name string, peerPublicKey string) (*driver.PeerStats, error) {
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

func (b *execBackend) FindForeignServers(ctx context.Context, knownInterfaces []string) ([]*driver.ForeignServer, error) {
	knownInterfaceSet := make(map[string]struct{}, len(knownInterfaces))
	for _, name := range knownInterfaces {
		knownInterfaceSet[name] = struct{}{}
	}

	existingForeignNames := make(map[string]struct{})

	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var foreignServers []*driver.ForeignServer
	for _, link := range links {
		if !strings.EqualFold(link.Type(), "wireguard") {
			continue
		}

		name := link.Attrs().Name
		if _, ok := knownInterfaceSet[name]; ok {
			continue
		}

		foreignInterface, err := netlinkInterfaceToForeignInterface(link)
		if err != nil {
			return nil, err
		}

		device, err := b.deviceFromDump(ctx, name)
		if err != nil {
			return nil, err
		}

		foreignServers = append(foreignServers, &driver.ForeignServer{
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

func (b *execBackend) syncWithoutRestart(ctx context.Context, options driver.ConfigureOptions, configPath string) error {
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

	if err := b.reconcileAddress(ctx, name, options.InterfaceOptions.Address); err != nil {
		return err
	}

	if err := b.reconcileMTU(ctx, name, options.InterfaceOptions.Mtu); err != nil {
		return err
	}

	if err := b.reconcileRoutes(ctx, name, options.WireguardOptions.Peers); err != nil {
		return err
	}

	return nil
}

func (b *execBackend) writeConfig(ctx context.Context, options driver.ConfigureOptions) (string, error) {
	name := options.InterfaceOptions.Name
	configPath := b.configFilePath(name)
	configContent := renderConfig(options)

	if b.useSudo {
		tmpPath, err := writeTempFile([]byte(configContent))
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpPath)

		if _, err := b.runCommand(ctx, "install", "-d", "-m", "0755", b.configDir); err != nil {
			return "", fmt.Errorf("failed to prepare config directory %s: %w", b.configDir, err)
		}
		if _, err := b.runCommand(ctx, "install", "-m", "0600", tmpPath, configPath); err != nil {
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

func (b *execBackend) deviceFromConfigFile(ctx context.Context, name string) (*driver.Device, error) {
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
) ([]*driver.ForeignServer, error) {
	configFiles, err := b.listConfigFiles(ctx)
	if err != nil {
		return nil, err
	}

	var foreignServers []*driver.ForeignServer
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
			logrus.WithError(err).
				WithField("path", configPath).
				Warn("failed to read foreign server config file")
			continue
		}

		parsedConfig, err := parseConfigDevice(name, string(configContent))
		if err != nil {
			logrus.WithError(err).
				WithField("path", configPath).
				Warn("failed to parse foreign server config file")
			continue
		}

		foreignServers = append(foreignServers, &driver.ForeignServer{
			Interface: &driver.ForeignInterface{
				Name:      parsedConfig.Device.Interface.Name,
				Addresses: parsedConfig.Device.Interface.Addresses,
				Mtu:       parsedConfig.Device.Interface.Mtu,
				State:     "down",
			},
			Hooks:        parsedConfig.Hooks,
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
		output, err := b.runCommand(ctx, "find", b.configDir, "-maxdepth", "1", "-type", "f", "-name", "*.conf", "-print")
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
		for i, file := range files {
			files[i] = strings.TrimSpace(file)
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
		output, err := b.runCommand(ctx, "cat", path)
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

func (b *execBackend) reconcileAddress(ctx context.Context, name string, desiredAddress string) error {
	state, err := b.getInterfaceState(ctx, name)
	if err != nil {
		return err
	}

	desired, err := normalizeCIDR(desiredAddress)
	if err != nil {
		return fmt.Errorf("failed to parse desired address %q: %w", desiredAddress, err)
	}

	current := make(map[string]struct{})
	for _, info := range state.AddrInfo {
		if info.Scope == "link" {
			continue
		}
		if info.Family != "inet" && info.Family != "inet6" {
			continue
		}
		if info.Local == "" || info.PrefixLen <= 0 {
			continue
		}

		addr, err := normalizeCIDR(fmt.Sprintf("%s/%d", info.Local, info.PrefixLen))
		if err != nil {
			continue
		}
		current[addr] = struct{}{}
	}

	if _, ok := current[desired]; !ok {
		if _, err := b.runIP(ctx, "address", "add", desired, "dev", name); err != nil {
			return fmt.Errorf("failed to add interface address %s: %w", desired, err)
		}
	}

	for addr := range current {
		if addr == desired {
			continue
		}
		if _, err := b.runIP(ctx, "address", "del", addr, "dev", name); err != nil {
			return fmt.Errorf("failed to remove interface address %s: %w", addr, err)
		}
	}

	return nil
}

func (b *execBackend) reconcileMTU(ctx context.Context, name string, desiredMTU int) error {
	if desiredMTU <= 0 {
		return nil
	}

	state, err := b.getInterfaceState(ctx, name)
	if err != nil {
		return err
	}

	if state.Mtu == desiredMTU {
		return nil
	}

	if _, err := b.runIP(ctx, "link", "set", "dev", name, "mtu", strconv.Itoa(desiredMTU)); err != nil {
		return fmt.Errorf("failed to set mtu to %d: %w", desiredMTU, err)
	}
	return nil
}

func (b *execBackend) reconcileRoutes(ctx context.Context, name string, peers []*driver.PeerOptions) error {
	desiredIPv4 := make(map[string]struct{})
	desiredIPv6 := make(map[string]struct{})

	for _, p := range peers {
		for _, allowedIP := range p.AllowedIPs {
			prefix, err := normalizeCIDR(allowedIP)
			if err != nil {
				return fmt.Errorf("failed to parse allowed ip %q: %w", allowedIP, err)
			}

			parsed, err := netip.ParsePrefix(prefix)
			if err != nil {
				return fmt.Errorf("failed to parse normalized allowed ip %q: %w", prefix, err)
			}

			if parsed.Addr().Is4() {
				desiredIPv4[prefix] = struct{}{}
			} else {
				desiredIPv6[prefix] = struct{}{}
			}
		}
	}

	for prefix := range desiredIPv4 {
		if _, err := b.runIP(ctx, "-4", "route", "replace", prefix, "dev", name); err != nil {
			return fmt.Errorf("failed to add or replace ipv4 route %s: %w", prefix, err)
		}
	}

	for prefix := range desiredIPv6 {
		if _, err := b.runIP(ctx, "-6", "route", "replace", prefix, "dev", name); err != nil {
			return fmt.Errorf("failed to add or replace ipv6 route %s: %w", prefix, err)
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

	for prefix := range currentIPv4 {
		if _, ok := desiredIPv4[prefix]; ok {
			continue
		}
		if _, err := b.runIP(ctx, "-4", "route", "del", prefix, "dev", name); err != nil {
			return fmt.Errorf("failed to remove ipv4 route %s: %w", prefix, err)
		}
	}

	for prefix := range currentIPv6 {
		if _, ok := desiredIPv6[prefix]; ok {
			continue
		}
		if _, err := b.runIP(ctx, "-6", "route", "del", prefix, "dev", name); err != nil {
			return fmt.Errorf("failed to remove ipv6 route %s: %w", prefix, err)
		}
	}

	return nil
}

func (b *execBackend) listRoutes(ctx context.Context, name string, ipv6 bool) (map[string]struct{}, error) {
	familyFlag := "-4"
	defaultPrefix := "0.0.0.0/0"
	if ipv6 {
		familyFlag = "-6"
		defaultPrefix = "::/0"
	}

	output, err := b.runIP(ctx, "-j", familyFlag, "route", "show", "dev", name)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes for %s: %w", name, err)
	}

	var routes []ipRouteState
	if err := json.Unmarshal(output, &routes); err != nil {
		return nil, fmt.Errorf("failed to decode route list: %w", err)
	}

	result := make(map[string]struct{})
	for _, route := range routes {
		dst := strings.TrimSpace(route.Dst)
		if dst == "" {
			continue
		}
		if dst == "default" {
			dst = defaultPrefix
		}
		prefix, err := normalizeCIDR(dst)
		if err != nil {
			continue
		}
		result[prefix] = struct{}{}
	}

	return result, nil
}

func (b *execBackend) getInterfaceState(ctx context.Context, name string) (*ipInterfaceState, error) {
	output, err := b.runIP(ctx, "-j", "address", "show", "dev", name)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface state for %s: %w", name, err)
	}

	var states []ipInterfaceState
	if err := json.Unmarshal(output, &states); err != nil {
		return nil, fmt.Errorf("failed to decode interface state for %s: %w", name, err)
	}
	if len(states) == 0 {
		return nil, fmt.Errorf("interface state for %s not found", name)
	}

	return &states[0], nil
}

func (b *execBackend) deviceFromDump(ctx context.Context, name string) (*driver.Device, error) {
	output, err := b.runWG(ctx, "show", name, "dump")
	if err != nil {
		return nil, fmt.Errorf("failed to read device dump for %s: %w", name, err)
	}
	return parseDeviceDump(name, string(output))
}

func (b *execBackend) interfaceExists(name string) (bool, error) {
	link, err := findInterface(name)
	if err != nil {
		return false, err
	}
	return link != nil, nil
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

func (b *execBackend) runIP(ctx context.Context, args ...string) ([]byte, error) {
	return b.runCommand(ctx, b.ipPath, args...)
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

func findInterface(name string) (netlink.Link, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		var linkNotFoundErr netlink.LinkNotFoundError
		if os.IsNotExist(err) || errors.As(err, &linkNotFoundErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find interface by name %s: %w", name, err)
	}
	return link, nil
}

func netlinkInterfaceToForeignInterface(link netlink.Link) (*driver.ForeignInterface, error) {
	attrs := link.Attrs()

	addrList, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get address list for interface %s", attrs.Name)
	}

	var addresses []string
	for _, addr := range addrList {
		addresses = append(addresses, addr.IPNet.String())
	}

	return &driver.ForeignInterface{
		Name:      attrs.Name,
		Addresses: addresses,
		Mtu:       attrs.MTU,
		State:     attrs.OperState.String(),
	}, nil
}

func linkStatisticsToBackendInterfaceStats(statistics *netlink.LinkStatistics) *driver.InterfaceStats {
	if statistics == nil {
		return nil
	}

	return &driver.InterfaceStats{
		RxBytes: statistics.RxBytes,
		TxBytes: statistics.TxBytes,
	}
}

func mapArray[T any, R any](items []T, fn func(T) R) []R {
	result := make([]R, 0, len(items))
	for _, item := range items {
		result = append(result, fn(item))
	}
	return result
}

type ipInterfaceState struct {
	Mtu      int             `json:"mtu"`
	AddrInfo []ipAddressInfo `json:"addr_info"`
}

type ipAddressInfo struct {
	Family    string `json:"family"`
	Local     string `json:"local"`
	PrefixLen int    `json:"prefixlen"`
	Scope     string `json:"scope"`
}

type ipRouteState struct {
	Dst string `json:"dst"`
}
