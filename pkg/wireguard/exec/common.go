package exec

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

const defaultConfigDir = "/etc/wireguard"

func writeTempFile(content []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "wg-ui-*.conf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	path := tmpFile.Name()

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(path)
	}

	if _, err := tmpFile.Write(content); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to write temporary file: %w", err)
	}
	if err := tmpFile.Chmod(0o600); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to chmod temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	return path, nil
}

func renderConfig(options driver.ConfigureOptions) string {
	interfaceOptions := options.InterfaceOptions
	wireguardOptions := options.WireguardOptions

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	if interfaceOptions.Description != "" {
		sb.WriteString("# ")
		sb.WriteString(strings.ReplaceAll(interfaceOptions.Description, "\n", " "))
		sb.WriteString("\n")
	}
	sb.WriteString("Address = ")
	sb.WriteString(interfaceOptions.Address)
	sb.WriteString("\n")
	sb.WriteString("PrivateKey = ")
	sb.WriteString(wireguardOptions.PrivateKey)
	sb.WriteString("\n")

	if wireguardOptions.ListenPort != nil {
		sb.WriteString("ListenPort = ")
		sb.WriteString(strconv.Itoa(*wireguardOptions.ListenPort))
		sb.WriteString("\n")
	}

	if wireguardOptions.FirewallMark != nil {
		sb.WriteString("FwMark = ")
		sb.WriteString(strconv.Itoa(*wireguardOptions.FirewallMark))
		sb.WriteString("\n")
	}

	if interfaceOptions.Mtu > 0 {
		sb.WriteString("MTU = ")
		sb.WriteString(strconv.Itoa(interfaceOptions.Mtu))
		sb.WriteString("\n")
	}

	for _, p := range wireguardOptions.Peers {
		sb.WriteString("\n[Peer]\n")
		sb.WriteString("PublicKey = ")
		sb.WriteString(p.PublicKey)
		sb.WriteString("\n")

		if p.PresharedKey != "" {
			sb.WriteString("PresharedKey = ")
			sb.WriteString(p.PresharedKey)
			sb.WriteString("\n")
		}

		if p.Endpoint != "" {
			sb.WriteString("Endpoint = ")
			sb.WriteString(p.Endpoint)
			sb.WriteString("\n")
		}

		if len(p.AllowedIPs) > 0 {
			sb.WriteString("AllowedIPs = ")
			sb.WriteString(strings.Join(p.AllowedIPs, ", "))
			sb.WriteString("\n")
		}

		if p.PersistentKeepalive > 0 {
			sb.WriteString("PersistentKeepalive = ")
			sb.WriteString(strconv.Itoa(p.PersistentKeepalive))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func parseConfigDevice(name string, content string) (*parsedConfigDevice, error) {
	parsed := &parsedConfigDevice{
		Device: &driver.Device{
			Interface: driver.Interface{
				Name: name,
			},
			Wireguard: driver.Wireguard{
				Name: name,
			},
		},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	section := ""
	var currentPeer *driver.Peer

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			if section == "interface" && parsed.Description == "" {
				parsed.Description = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			}
			continue
		}

		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		switch strings.ToLower(line) {
		case "[interface]":
			section = "interface"
			currentPeer = nil
			continue
		case "[peer]":
			section = "peer"
			currentPeer = &driver.Peer{}
			parsed.Device.Wireguard.Peers = append(parsed.Device.Wireguard.Peers, currentPeer)
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		switch section {
		case "interface":
			switch key {
			case "address":
				var addresses []string
				for _, address := range strings.Split(value, ",") {
					address = strings.TrimSpace(address)
					if address == "" {
						continue
					}
					if _, err := normalizeCIDR(address); err != nil {
						return nil, fmt.Errorf("invalid interface address %q: %w", address, err)
					}
					addresses = append(addresses, address)
				}
				parsed.Device.Interface.Addresses = addresses
			case "privatekey":
				parsed.Device.Wireguard.PrivateKey = value
			case "listenport":
				listenPort, err := parseConfigInt(value)
				if err != nil {
					return nil, fmt.Errorf("invalid listen port %q: %w", value, err)
				}
				parsed.Device.Wireguard.ListenPort = listenPort
			case "fwmark":
				firewallMark, err := parseConfigInt(value)
				if err != nil {
					return nil, fmt.Errorf("invalid fwmark %q: %w", value, err)
				}
				parsed.Device.Wireguard.FirewallMark = firewallMark
			case "mtu":
				mtu, err := parseConfigInt(value)
				if err != nil {
					return nil, fmt.Errorf("invalid mtu %q: %w", value, err)
				}
				parsed.Device.Interface.Mtu = mtu
			}
		case "peer":
			if currentPeer == nil {
				continue
			}
			switch key {
			case "publickey":
				currentPeer.PublicKey = value
			case "presharedkey":
				currentPeer.PresharedKey = value
			case "endpoint":
				currentPeer.Endpoint = value
			case "allowedips":
				var allowedIPs []net.IPNet
				for _, allowedIP := range strings.Split(value, ",") {
					allowedIP = strings.TrimSpace(allowedIP)
					if allowedIP == "" {
						continue
					}
					_, ipNet, err := net.ParseCIDR(allowedIP)
					if err != nil {
						return nil, fmt.Errorf("invalid peer allowed ip %q: %w", allowedIP, err)
					}
					allowedIPs = append(allowedIPs, *ipNet)
				}
				currentPeer.AllowedIPs = allowedIPs
			case "persistentkeepalive":
				persistentKeepalive, err := parseConfigInt(value)
				if err != nil {
					return nil, fmt.Errorf("invalid persistent keepalive %q: %w", value, err)
				}
				currentPeer.PersistentKeepalive = time.Duration(persistentKeepalive) * time.Second
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan config content: %w", err)
	}

	if parsed.Device.Wireguard.PrivateKey != "" {
		privateKey, err := wgtypes.ParseKey(parsed.Device.Wireguard.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key in config: %w", err)
		}
		parsed.Device.Wireguard.PublicKey = privateKey.PublicKey().String()
	}

	return parsed, nil
}

func parseDeviceDump(name string, output string) (*driver.Device, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, errors.New("wireguard dump output is empty")
	}

	lines := strings.Split(trimmed, "\n")
	interfaceFields := splitDumpFields(lines[0])
	if len(interfaceFields) < 4 {
		return nil, fmt.Errorf("invalid interface dump line: %q", lines[0])
	}

	listenPort, err := parseDumpInt(interfaceFields[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse listen port: %w", err)
	}
	firewallMark, err := parseDumpInt(interfaceFields[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse firewall mark: %w", err)
	}

	device := &driver.Device{
		Wireguard: driver.Wireguard{
			Name:         name,
			PrivateKey:   parseDumpString(interfaceFields[0]),
			PublicKey:    parseDumpString(interfaceFields[1]),
			ListenPort:   listenPort,
			FirewallMark: firewallMark,
		},
	}

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		peerFields := splitDumpFields(line)
		if len(peerFields) < 8 {
			return nil, fmt.Errorf("invalid peer dump line: %q", line)
		}

		latestHandshakeUnix, err := strconv.ParseInt(peerFields[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse latest handshake for peer %s: %w", peerFields[0], err)
		}

		receiveBytes, err := parseDumpInt64(peerFields[5])
		if err != nil {
			return nil, fmt.Errorf("failed to parse receive bytes for peer %s: %w", peerFields[0], err)
		}

		transmitBytes, err := parseDumpInt64(peerFields[6])
		if err != nil {
			return nil, fmt.Errorf("failed to parse transmit bytes for peer %s: %w", peerFields[0], err)
		}

		persistentKeepalive, err := parseDumpInt(peerFields[7])
		if err != nil {
			return nil, fmt.Errorf("failed to parse keepalive for peer %s: %w", peerFields[0], err)
		}

		var allowedIPs []net.IPNet
		allowedIPsField := strings.TrimSpace(peerFields[3])
		if allowedIPsField != "" && allowedIPsField != "(none)" {
			for _, allowedIP := range strings.Split(allowedIPsField, ",") {
				allowedIP = strings.TrimSpace(allowedIP)
				_, ipNet, err := net.ParseCIDR(allowedIP)
				if err != nil {
					return nil, fmt.Errorf("failed to parse allowed ip %q for peer %s: %w", allowedIP, peerFields[0], err)
				}
				allowedIPs = append(allowedIPs, *ipNet)
			}
		}

		endpoint := parseDumpString(peerFields[2])
		presharedKey := parseDumpString(peerFields[1])

		var latestHandshake time.Time
		if latestHandshakeUnix > 0 {
			latestHandshake = time.Unix(latestHandshakeUnix, 0)
		}

		device.Wireguard.Peers = append(device.Wireguard.Peers, &driver.Peer{
			PublicKey:           parseDumpString(peerFields[0]),
			Endpoint:            endpoint,
			AllowedIPs:          allowedIPs,
			PresharedKey:        presharedKey,
			PersistentKeepalive: time.Duration(persistentKeepalive) * time.Second,
			Stats: driver.PeerStats{
				LastHandshakeTime: latestHandshake,
				ReceiveBytes:      receiveBytes,
				TransmitBytes:     transmitBytes,
			},
		})
	}

	return device, nil
}

func splitDumpFields(line string) []string {
	fields := strings.Split(line, "\t")
	if len(fields) == 1 {
		fields = strings.Fields(line)
	}
	return fields
}

func parseDumpString(v string) string {
	v = strings.TrimSpace(v)
	if v == "(none)" {
		return ""
	}
	return v
}

func parseDumpInt(v string) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" || v == "off" || v == "(none)" {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func parseConfigInt(v string) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "off") || strings.EqualFold(v, "(none)") {
		return 0, nil
	}

	n, err := strconv.ParseInt(v, 0, 64)
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func parseDumpInt64(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" || v == "off" || v == "(none)" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return n, nil
	}

	u, uErr := strconv.ParseUint(v, 10, 64)
	if uErr != nil {
		return 0, err
	}
	if u > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(u), nil
}

func normalizeCIDR(cidr string) (string, error) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
	if err != nil {
		return "", err
	}
	return prefix.Masked().String(), nil
}

type parsedConfigDevice struct {
	Description string
	Device      *driver.Device
}
