package exec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestRenderConfigIncludesInterfaceAndPeerSettings(t *testing.T) {
	listenPort := 51820
	firewallMark := 42
	config := renderConfig(backend.ConfigureOptions{
		InterfaceOptions: backend.InterfaceOptions{
			Name:        "wg0",
			Description: "My tunnel",
			Address:     "10.0.0.1/24",
			Mtu:         1420,
		},
		WireguardOptions: backend.WireguardOptions{
			PrivateKey:   "private-key",
			ListenPort:   &listenPort,
			FirewallMark: &firewallMark,
			Peers: []*backend.PeerOptions{
				{
					PublicKey:           "peer-public",
					Endpoint:            "198.51.100.10:51820",
					AllowedIPs:          []string{"10.0.0.2/32", "fd00::2/128"},
					PresharedKey:        "peer-psk",
					PersistentKeepalive: 25,
				},
			},
		},
	})

	for _, fragment := range []string{
		"[Interface]",
		"# My tunnel",
		"Address = 10.0.0.1/24",
		"PrivateKey = private-key",
		"ListenPort = 51820",
		"FwMark = 42",
		"MTU = 1420",
		"[Peer]",
		"PublicKey = peer-public",
		"PresharedKey = peer-psk",
		"Endpoint = 198.51.100.10:51820",
		"AllowedIPs = 10.0.0.2/32, fd00::2/128",
		"PersistentKeepalive = 25",
	} {
		if !strings.Contains(config, fragment) {
			t.Fatalf("expected config to contain %q, got:\n%s", fragment, config)
		}
	}
}

func TestParseDeviceDump(t *testing.T) {
	output := strings.Join([]string{
		"private-key\tpublic-key\t51820\toff",
		"peer-key\tpeer-psk\t203.0.113.10:51820\t10.0.0.2/32,fd00::2/128\t1700000000\t123\t456\t25",
	}, "\n")

	device, err := parseDeviceDump("wg0", output)
	if err != nil {
		t.Fatalf("parseDeviceDump returned error: %v", err)
	}

	if device.Wireguard.Name != "wg0" {
		t.Fatalf("expected device name wg0, got %q", device.Wireguard.Name)
	}
	if device.Wireguard.PrivateKey != "private-key" {
		t.Fatalf("unexpected private key: %q", device.Wireguard.PrivateKey)
	}
	if device.Wireguard.PublicKey != "public-key" {
		t.Fatalf("unexpected public key: %q", device.Wireguard.PublicKey)
	}
	if device.Wireguard.ListenPort != 51820 {
		t.Fatalf("unexpected listen port: %d", device.Wireguard.ListenPort)
	}
	if device.Wireguard.FirewallMark != 0 {
		t.Fatalf("expected firewall mark 0 for off, got %d", device.Wireguard.FirewallMark)
	}
	if len(device.Wireguard.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(device.Wireguard.Peers))
	}

	peer := device.Wireguard.Peers[0]
	if peer.PublicKey != "peer-key" {
		t.Fatalf("unexpected peer public key: %q", peer.PublicKey)
	}
	if peer.Endpoint != "203.0.113.10:51820" {
		t.Fatalf("unexpected peer endpoint: %q", peer.Endpoint)
	}
	if len(peer.AllowedIPs) != 2 {
		t.Fatalf("expected 2 allowed IPs, got %d", len(peer.AllowedIPs))
	}
	if peer.PersistentKeepalive != 25*time.Second {
		t.Fatalf("unexpected keepalive: %s", peer.PersistentKeepalive)
	}
	if peer.Stats.LastHandshakeTime.Unix() != 1700000000 {
		t.Fatalf("unexpected last handshake time: %d", peer.Stats.LastHandshakeTime.Unix())
	}
	if peer.Stats.ReceiveBytes != 123 {
		t.Fatalf("unexpected rx bytes: %d", peer.Stats.ReceiveBytes)
	}
	if peer.Stats.TransmitBytes != 456 {
		t.Fatalf("unexpected tx bytes: %d", peer.Stats.TransmitBytes)
	}
}

func TestRunCommandSudoPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	fakeSudoPath := tmpDir + "/fake-sudo"
	if err := os.WriteFile(fakeSudoPath, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o755); err != nil {
		t.Fatalf("failed to create fake sudo command: %v", err)
	}

	backend := &execBackend{
		useSudo:  true,
		sudoPath: fakeSudoPath,
	}

	output, err := backend.runCommand(context.Background(), "wg", "show")
	if err != nil {
		t.Fatalf("runCommand returned error: %v", err)
	}

	if strings.TrimSpace(string(output)) != "-n\nwg\nshow" {
		t.Fatalf("expected sudo-prefixed command args, got %q", strings.TrimSpace(string(output)))
	}
}

func TestParseConfigDevice(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	configContent := fmt.Sprintf(`[Interface]
# Imported config
Address = 10.10.0.1/24, fd10::1/64
PrivateKey = %s
ListenPort = 51821
FwMark = 0x2a
MTU = 1420

[Peer]
PublicKey = peer-public-key
PresharedKey = peer-psk
Endpoint = 203.0.113.5:51820
AllowedIPs = 10.10.0.2/32, fd10::2/128
PersistentKeepalive = 20
`, privateKey.String())

	parsed, err := parseConfigDevice("wg10", configContent)
	if err != nil {
		t.Fatalf("parseConfigDevice returned error: %v", err)
	}

	if parsed.Description != "Imported config" {
		t.Fatalf("expected description to be parsed, got %q", parsed.Description)
	}
	if parsed.Device.Interface.Name != "wg10" {
		t.Fatalf("expected interface name wg10, got %q", parsed.Device.Interface.Name)
	}
	if len(parsed.Device.Interface.Addresses) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(parsed.Device.Interface.Addresses))
	}
	if parsed.Device.Interface.Mtu != 1420 {
		t.Fatalf("expected mtu 1420, got %d", parsed.Device.Interface.Mtu)
	}
	if parsed.Device.Wireguard.PrivateKey != privateKey.String() {
		t.Fatalf("unexpected private key")
	}
	if parsed.Device.Wireguard.PublicKey != privateKey.PublicKey().String() {
		t.Fatalf("unexpected derived public key")
	}
	if parsed.Device.Wireguard.ListenPort != 51821 {
		t.Fatalf("expected listen port 51821, got %d", parsed.Device.Wireguard.ListenPort)
	}
	if parsed.Device.Wireguard.FirewallMark != 42 {
		t.Fatalf("expected fwmark 42, got %d", parsed.Device.Wireguard.FirewallMark)
	}
	if len(parsed.Device.Wireguard.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(parsed.Device.Wireguard.Peers))
	}

	peer := parsed.Device.Wireguard.Peers[0]
	if peer.PersistentKeepalive != 20*time.Second {
		t.Fatalf("expected keepalive 20s, got %s", peer.PersistentKeepalive)
	}
	if len(peer.AllowedIPs) != 2 {
		t.Fatalf("expected 2 allowed ips, got %d", len(peer.AllowedIPs))
	}
}

func TestDeviceFallsBackToConfigFileWhenInterfaceIsDown(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	tmpDir := t.TempDir()
	name := "wg-ui-offline-test"
	configPath := filepath.Join(tmpDir, name+".conf")
	configContent := fmt.Sprintf(`[Interface]
Address = 10.20.0.1/24
PrivateKey = %s
ListenPort = 51820
`, privateKey.String())
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	execBackend := &execBackend{
		configDir: tmpDir,
		wgPath:    "/command/that/does/not/exist",
	}

	device, err := execBackend.Device(context.Background(), name)
	if err != nil {
		t.Fatalf("Device returned error: %v", err)
	}
	if device.Wireguard.PublicKey != privateKey.PublicKey().String() {
		t.Fatalf("expected public key from config, got %q", device.Wireguard.PublicKey)
	}
	if device.Wireguard.ListenPort != 51820 {
		t.Fatalf("expected listen port 51820, got %d", device.Wireguard.ListenPort)
	}
	if len(device.Interface.Addresses) != 1 || device.Interface.Addresses[0] != "10.20.0.1/24" {
		t.Fatalf("unexpected addresses: %#v", device.Interface.Addresses)
	}
}
