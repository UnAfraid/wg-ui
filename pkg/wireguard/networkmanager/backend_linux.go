//go:build linux

package networkmanager

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/Wifx/gonetworkmanager/v3"
	"github.com/google/uuid"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

func init() {
	supported := isNetworkManagerAvailable()
	backend.Register("networkmanager", NewNetworkManagerBackend, supported)
}

func isNetworkManagerAvailable() bool {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return false
	}
	_, err = nm.GetPropertyVersion()
	return err == nil
}

type nmBackend struct {
	nm       gonetworkmanager.NetworkManager
	settings gonetworkmanager.Settings
	wgClient *wgctrl.Client
}

func NewNetworkManagerBackend() (backend.Backend, error) {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NetworkManager: %w", err)
	}

	settings, err := gonetworkmanager.NewSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NetworkManager settings: %w", err)
	}

	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize wireguard client: %w", err)
	}

	return &nmBackend{
		nm:       nm,
		settings: settings,
		wgClient: wgClient,
	}, nil
}

func (b *nmBackend) Device(ctx context.Context, name string) (*backend.Device, error) {
	conn, err := b.findConnectionByInterfaceName(name)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, nil
	}

	device, err := b.wgClient.Device(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get wireguard device: %w", err)
	}

	return b.buildDevice(conn, device)
}

func (b *nmBackend) Up(ctx context.Context, options backend.ConfigureOptions) (*backend.Device, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	interfaceOpts := options.InterfaceOptions
	wireguardOpts := options.WireguardOptions

	conn, err := b.findConnectionByInterfaceName(interfaceOpts.Name)
	if err != nil {
		return nil, err
	}

	connectionSettings := b.buildConnectionSettings(interfaceOpts, wireguardOpts)

	if conn == nil {
		conn, err = b.settings.AddConnection(connectionSettings)
		if err != nil {
			return nil, fmt.Errorf("failed to add connection: %w", err)
		}
	} else {
		if err := conn.Update(connectionSettings); err != nil {
			return nil, fmt.Errorf("failed to update connection: %w", err)
		}
	}

	nmDevice, err := b.nm.GetDeviceByIpIface(interfaceOpts.Name)
	if err != nil {
		_, err = b.nm.ActivateConnection(conn, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to activate connection: %w", err)
		}

		// Wait for the device to become available
		for i := 0; i < 10; i++ {
			nmDevice, err = b.nm.GetDeviceByIpIface(interfaceOpts.Name)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			return nil, fmt.Errorf("device not available after activation: %w", err)
		}
	} else {
		state, err := nmDevice.GetPropertyState()
		if err != nil {
			return nil, fmt.Errorf("failed to get device state: %w", err)
		}
		if state != gonetworkmanager.NmDeviceStateActivated {
			_, err = b.nm.ActivateConnection(conn, nmDevice, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to activate connection: %w", err)
			}
		}
	}

	// Get the wireguard device info
	device, err := b.wgClient.Device(interfaceOpts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get wireguard device: %w", err)
	}

	return b.buildDevice(conn, device)
}

func (b *nmBackend) Down(ctx context.Context, name string) error {
	conn, err := b.findConnectionByInterfaceName(name)
	if err != nil {
		return err
	}
	if conn == nil {
		return nil
	}

	nmDevice, err := b.nm.GetDeviceByIpIface(name)
	if err == nil {
		activeConn, err := nmDevice.GetPropertyActiveConnection()
		if err == nil && activeConn != nil {
			if err := b.nm.DeactivateConnection(activeConn); err != nil {
				return fmt.Errorf("failed to deactivate connection: %w", err)
			}
		}
	}

	if err := conn.Delete(); err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

func (b *nmBackend) Status(ctx context.Context, name string) (bool, error) {
	nmDevice, err := b.nm.GetDeviceByIpIface(name)
	if err != nil {
		return false, nil
	}

	state, err := nmDevice.GetPropertyState()
	if err != nil {
		return false, fmt.Errorf("failed to get device state: %w", err)
	}

	return state == gonetworkmanager.NmDeviceStateActivated, nil
}

func (b *nmBackend) Stats(ctx context.Context, name string) (*backend.InterfaceStats, error) {
	nmDevice, err := b.nm.GetDeviceByIpIface(name)
	if err != nil {
		return nil, nil
	}

	stats, err := gonetworkmanager.NewDeviceStatistics(nmDevice.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to get device statistics: %w", err)
	}

	rxBytes, _ := stats.GetPropertyRxBytes()
	txBytes, _ := stats.GetPropertyTxBytes()

	return &backend.InterfaceStats{
		RxBytes: rxBytes,
		TxBytes: txBytes,
	}, nil
}

func (b *nmBackend) PeerStats(ctx context.Context, name string, peerPublicKey string) (*backend.PeerStats, error) {
	device, err := b.wgClient.Device(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get wireguard device: %w", err)
	}

	publicKey, err := wgtypes.ParseKey(peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid peer public key: %w", err)
	}

	for _, peer := range device.Peers {
		if peer.PublicKey == publicKey {
			return &backend.PeerStats{
				LastHandshakeTime: peer.LastHandshakeTime,
				ReceiveBytes:      peer.ReceiveBytes,
				TransmitBytes:     peer.TransmitBytes,
				ProtocolVersion:   peer.ProtocolVersion,
			}, nil
		}
	}

	return nil, nil
}

func (b *nmBackend) FindForeignServers(ctx context.Context, knownInterfaces []string) ([]*backend.ForeignServer, error) {
	devices, err := b.nm.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	var foreignServers []*backend.ForeignServer
	for _, nmDevice := range devices {
		deviceType, err := nmDevice.GetPropertyDeviceType()
		if err != nil {
			continue
		}

		if deviceType != gonetworkmanager.NmDeviceTypeWireguard {
			continue
		}

		interfaceName, err := nmDevice.GetPropertyInterface()
		if err != nil {
			continue
		}

		if slices.Contains(knownInterfaces, interfaceName) {
			continue
		}

		foreignServer, err := b.buildForeignServer(nmDevice)
		if err != nil {
			continue
		}

		foreignServers = append(foreignServers, foreignServer)
	}

	return foreignServers, nil
}

func (b *nmBackend) Close(ctx context.Context) error {
	return b.wgClient.Close()
}

func (b *nmBackend) Supported() bool {
	return true
}

func (b *nmBackend) findConnectionByInterfaceName(name string) (gonetworkmanager.Connection, error) {
	connections, err := b.settings.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	for _, conn := range connections {
		settings, err := conn.GetSettings()
		if err != nil {
			continue
		}

		connSettings, ok := settings["connection"]
		if !ok {
			continue
		}

		connType, ok := connSettings["type"].(string)
		if !ok || connType != "wireguard" {
			continue
		}

		interfaceName, ok := connSettings["interface-name"].(string)
		if ok && interfaceName == name {
			return conn, nil
		}
	}

	return nil, nil
}

func (b *nmBackend) buildConnectionSettings(interfaceOpts backend.InterfaceOptions, wireguardOpts backend.WireguardOptions) gonetworkmanager.ConnectionSettings {
	connectionUUID := uuid.New().String()

	settings := gonetworkmanager.ConnectionSettings{
		"connection": {
			"id":             interfaceOpts.Name,
			"uuid":           connectionUUID,
			"type":           "wireguard",
			"interface-name": interfaceOpts.Name,
			"autoconnect":    false,
		},
		"wireguard": {
			"private-key": wireguardOpts.PrivateKey,
		},
		"ipv4": {
			"method": "manual",
		},
		"ipv6": {
			"method": "ignore",
		},
	}

	if wireguardOpts.ListenPort != nil {
		settings["wireguard"]["listen-port"] = uint32(*wireguardOpts.ListenPort)
	}

	if wireguardOpts.FirewallMark != nil {
		settings["wireguard"]["fwmark"] = uint32(*wireguardOpts.FirewallMark)
	}

	// Parse address and set up IPv4/IPv6 config
	addr := interfaceOpts.Address
	if strings.Contains(addr, "/") {
		ip, ipNet, err := net.ParseCIDR(addr)
		if err == nil {
			prefixLen, _ := ipNet.Mask.Size()
			if ip.To4() != nil {
				settings["ipv4"]["address-data"] = []map[string]interface{}{
					{
						"address": ip.String(),
						"prefix":  uint32(prefixLen),
					},
				}
			} else {
				settings["ipv6"]["method"] = "manual"
				settings["ipv6"]["address-data"] = []map[string]interface{}{
					{
						"address": ip.String(),
						"prefix":  uint32(prefixLen),
					},
				}
			}
		}
	}

	if interfaceOpts.Mtu > 0 {
		settings["wireguard"]["mtu"] = uint32(interfaceOpts.Mtu)
	}

	// Add peers
	if len(wireguardOpts.Peers) > 0 {
		var peers []map[string]interface{}
		for _, peer := range wireguardOpts.Peers {
			peerConfig := map[string]interface{}{
				"public-key": peer.PublicKey,
			}

			if peer.PresharedKey != "" {
				peerConfig["preshared-key"] = peer.PresharedKey
				peerConfig["preshared-key-flags"] = uint32(0)
			}

			if peer.Endpoint != "" {
				peerConfig["endpoint"] = peer.Endpoint
			}

			if peer.PersistentKeepalive > 0 {
				peerConfig["persistent-keepalive"] = uint32(peer.PersistentKeepalive)
			}

			if len(peer.AllowedIPs) > 0 {
				peerConfig["allowed-ips"] = peer.AllowedIPs
			}

			peers = append(peers, peerConfig)
		}
		settings["wireguard"]["peers"] = peers
	}

	return settings
}

func (b *nmBackend) buildDevice(conn gonetworkmanager.Connection, device *wgtypes.Device) (*backend.Device, error) {
	settings, err := conn.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection settings: %w", err)
	}

	var addresses []string
	if ipv4Settings, ok := settings["ipv4"]; ok {
		if addrData, ok := ipv4Settings["addresses"].([][]interface{}); ok {
			for _, addr := range addrData {
				if len(addr) >= 2 {
					ip := fmt.Sprintf("%v", addr[0])
					prefix := addr[1]
					addresses = append(addresses, fmt.Sprintf("%s/%v", ip, prefix))
				}
			}
		}
		if addrData, ok := ipv4Settings["address-data"].([]map[string]interface{}); ok {
			for _, addr := range addrData {
				if addrStr, ok := addr["address"].(string); ok {
					prefix := uint32(24)
					if p, ok := addr["prefix"].(uint32); ok {
						prefix = p
					}
					addresses = append(addresses, fmt.Sprintf("%s/%d", addrStr, prefix))
				}
			}
		}
	}
	if ipv6Settings, ok := settings["ipv6"]; ok {
		if addrData, ok := ipv6Settings["addresses"].([][]interface{}); ok {
			for _, addr := range addrData {
				if len(addr) >= 2 {
					ip := fmt.Sprintf("%v", addr[0])
					prefix := addr[1]
					addresses = append(addresses, fmt.Sprintf("%s/%v", ip, prefix))
				}
			}
		}
	}

	var mtu int
	if wgSettings, ok := settings["wireguard"]; ok {
		if m, ok := wgSettings["mtu"].(uint32); ok {
			mtu = int(m)
		}
	}

	return &backend.Device{
		Interface: backend.Interface{
			Name:      device.Name,
			Addresses: addresses,
			Mtu:       mtu,
		},
		Wireguard: backend.Wireguard{
			Name:         device.Name,
			PublicKey:    device.PublicKey.String(),
			PrivateKey:   device.PrivateKey.String(),
			ListenPort:   device.ListenPort,
			FirewallMark: device.FirewallMark,
			Peers: adapt.Array(device.Peers, func(peer wgtypes.Peer) *backend.Peer {
				var endpoint string
				if peer.Endpoint != nil {
					endpoint = peer.Endpoint.String()
				}
				return &backend.Peer{
					PublicKey:           peer.PublicKey.String(),
					Endpoint:            endpoint,
					AllowedIPs:          peer.AllowedIPs,
					PresharedKey:        peer.PresharedKey.String(),
					PersistentKeepalive: peer.PersistentKeepaliveInterval,
					Stats: backend.PeerStats{
						LastHandshakeTime: peer.LastHandshakeTime,
						ReceiveBytes:      peer.ReceiveBytes,
						TransmitBytes:     peer.TransmitBytes,
						ProtocolVersion:   peer.ProtocolVersion,
					},
				}
			}),
		},
	}, nil
}

func (b *nmBackend) buildForeignServer(nmDevice gonetworkmanager.Device) (*backend.ForeignServer, error) {
	interfaceName, err := nmDevice.GetPropertyInterface()
	if err != nil {
		return nil, err
	}

	device, err := b.wgClient.Device(interfaceName)
	if err != nil {
		return nil, err
	}

	mtu, _ := nmDevice.GetPropertyMtu()
	state, _ := nmDevice.GetPropertyState()

	var addresses []string
	ip4Config, err := nmDevice.GetPropertyIP4Config()
	if err == nil && ip4Config != nil {
		addrData, err := ip4Config.GetPropertyAddressData()
		if err == nil {
			for _, addr := range addrData {
				addresses = append(addresses, fmt.Sprintf("%s/%d", addr.Address, addr.Prefix))
			}
		}
	}

	return &backend.ForeignServer{
		Interface: &backend.ForeignInterface{
			Name:      interfaceName,
			Addresses: addresses,
			Mtu:       int(mtu),
			State:     state.String(),
		},
		Name:         device.Name,
		Type:         device.Type.String(),
		PublicKey:    device.PublicKey.String(),
		ListenPort:   device.ListenPort,
		FirewallMark: device.FirewallMark,
		Peers: adapt.Array(device.Peers, func(peer wgtypes.Peer) *backend.Peer {
			var endpoint string
			if peer.Endpoint != nil {
				endpoint = peer.Endpoint.String()
			}
			return &backend.Peer{
				PublicKey:           peer.PublicKey.String(),
				Endpoint:            endpoint,
				AllowedIPs:          peer.AllowedIPs,
				PresharedKey:        peer.PresharedKey.String(),
				PersistentKeepalive: peer.PersistentKeepaliveInterval,
				Stats: backend.PeerStats{
					LastHandshakeTime: peer.LastHandshakeTime,
					ReceiveBytes:      peer.ReceiveBytes,
					TransmitBytes:     peer.TransmitBytes,
					ProtocolVersion:   peer.ProtocolVersion,
				},
			}
		}),
	}, nil
}
