//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

func init() {
	backend.Register("darwin", NewDarwinBackend, true)
}

type darwinBackend struct {
	client  *wgctrl.Client
	devices map[string]*managedDevice // keyed by requested name
	mu      sync.RWMutex
}

type managedDevice struct {
	name         string
	device       *device.Device
	tunDevice    tun.Device
	uapiListener net.Listener
	logger       *device.Logger
}

func NewDarwinBackend() (backend.Backend, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize darwin backend: %w", err)
	}

	return &darwinBackend{
		client:  client,
		devices: make(map[string]*managedDevice),
	}, nil
}

func (db *darwinBackend) Device(_ context.Context, name string) (*backend.Device, error) {
	db.mu.RLock()
	managed, exists := db.devices[name]
	db.mu.RUnlock()

	actualName := name
	if exists {
		actualName = managed.name
	}

	device, err := db.client.Device(actualName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return wgDeviceToBackendDevice(device, actualName)
}

func (db *darwinBackend) Up(_ context.Context, options backend.ConfigureOptions) (*backend.Device, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	interfaceOptions := options.InterfaceOptions
	wireguardOptions := options.WireguardOptions

	// Use original name as the lookup key (not normalized)
	requestedName := interfaceOptions.Name

	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if device already exists by requested name
	if _, exists := db.devices[requestedName]; exists {
		managed := db.devices[requestedName]
		// Update existing device using its actual kernel name
		interfaceOptions.Name = managed.name
		return db.updateDevice(interfaceOptions, wireguardOptions)
	}

	// Create new device
	return db.createDevice(requestedName, interfaceOptions, wireguardOptions)
}

func (db *darwinBackend) createDevice(requestedName string, interfaceOpts backend.InterfaceOptions, wireguardOpts backend.WireguardOptions) (*backend.Device, error) {
	// On macOS, we MUST use "utun" (no number) to let the kernel assign the next available number.
	// If we specify "utun0", "utun1", etc., it will fail if that specific interface exists,
	// even if it's not being used by WireGuard. The kernel manages utun numbering automatically.
	tunName := "utun"

	// Create TUN device - kernel will assign next available utun number
	tunDevice, err := tun.CreateTUN(tunName, interfaceOpts.Mtu)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device: %w", err)
	}

	// Get actual interface name assigned by kernel (e.g., "utun9", "utun10", etc.)
	actualName, err := tunDevice.Name()
	if err != nil {
		tunDevice.Close()
		return nil, fmt.Errorf("failed to get TUN device name: %w", err)
	}

	// Log the actual interface name that was created
	fmt.Fprintf(os.Stderr, "Created TUN device: requested=%s, actual=%s\n", interfaceOpts.Name, actualName)

	// Create logger
	logger := device.NewLogger(
		device.LogLevelError,
		fmt.Sprintf("(%s) ", actualName),
	)

	// Create WireGuard device
	wgDevice := device.NewDevice(tunDevice, conn.NewDefaultBind(), logger)

	// Create UAPI socket so wgctrl can communicate with the device
	// This creates /var/run/wireguard/<name>.sock
	fileUAPI, err := ipc.UAPIOpen(actualName)
	if err != nil {
		wgDevice.Close()
		return nil, fmt.Errorf("failed to open UAPI socket: %w", err)
	}

	uapiListener, err := ipc.UAPIListen(actualName, fileUAPI)
	if err != nil {
		fileUAPI.Close()
		wgDevice.Close()
		return nil, fmt.Errorf("failed to listen on UAPI socket: %w", err)
	}

	// Handle UAPI connections in background
	go func() {
		for {
			uapiConn, err := uapiListener.Accept()
			if err != nil {
				return
			}
			go wgDevice.IpcHandle(uapiConn)
		}
	}()

	// Bring device up
	if err := wgDevice.Up(); err != nil {
		uapiListener.Close()
		wgDevice.Close()
		return nil, fmt.Errorf("failed to bring device up: %w", err)
	}

	// Configure network interface (IP address, MTU)
	if err := configureInterface(actualName, interfaceOpts.Address, interfaceOpts.Mtu); err != nil {
		uapiListener.Close()
		wgDevice.Close()
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}

	// Configure WireGuard (keys, peers) - requires UAPI socket
	if err := db.configureWireguard(actualName, wireguardOpts); err != nil {
		uapiListener.Close()
		wgDevice.Close()
		return nil, fmt.Errorf("failed to configure wireguard: %w", err)
	}

	// Add routes for peers
	if err := configureRoutes(actualName, wireguardOpts.Peers); err != nil {
		uapiListener.Close()
		wgDevice.Close()
		return nil, fmt.Errorf("failed to configure routes: %w", err)
	}

	// Store managed device keyed by the requested name
	db.devices[requestedName] = &managedDevice{
		name:         actualName,
		device:       wgDevice,
		tunDevice:    tunDevice,
		uapiListener: uapiListener,
		logger:       logger,
	}

	// Get device info to return
	wgInfo, err := db.client.Device(actualName)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	return wgDeviceToBackendDevice(wgInfo, actualName)
}

func (db *darwinBackend) updateDevice(interfaceOpts backend.InterfaceOptions, wireguardOpts backend.WireguardOptions) (*backend.Device, error) {
	name := interfaceOpts.Name

	// Configure network interface
	if err := configureInterface(name, interfaceOpts.Address, interfaceOpts.Mtu); err != nil {
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}

	// Configure WireGuard
	if err := db.configureWireguard(name, wireguardOpts); err != nil {
		return nil, fmt.Errorf("failed to configure wireguard: %w", err)
	}

	// Update routes
	if err := configureRoutes(name, wireguardOpts.Peers); err != nil {
		return nil, fmt.Errorf("failed to configure routes: %w", err)
	}

	// Get device info to return
	wgInfo, err := db.client.Device(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	return wgDeviceToBackendDevice(wgInfo, name)
}

func (db *darwinBackend) Down(_ context.Context, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	managed, exists := db.devices[name]
	if !exists {
		return nil
	}

	// Remove routes using actual kernel name
	if err := removeRoutes(managed.name); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove routes for %s: %v\n", managed.name, err)
	}

	// Close UAPI listener
	if managed.uapiListener != nil {
		managed.uapiListener.Close()
	}

	// Close WireGuard device (also closes TUN device)
	managed.device.Close()

	// Remove from managed devices
	delete(db.devices, name)

	return nil
}

func (db *darwinBackend) Status(_ context.Context, name string) (bool, error) {
	// Check if we're managing this device
	db.mu.RLock()
	managed, exists := db.devices[name]
	db.mu.RUnlock()

	if exists {
		select {
		case <-managed.device.Wait():
			return false, nil
		default:
			return true, nil
		}
	}

	// Not managed by us, check via wgctrl
	_, err := db.client.Device(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check device status: %w", err)
	}

	return true, nil
}

func (db *darwinBackend) Stats(_ context.Context, name string) (*backend.InterfaceStats, error) {
	db.mu.RLock()
	managed, exists := db.devices[name]
	db.mu.RUnlock()

	actualName := name
	if exists {
		actualName = managed.name
	}
	return interfaceStats(actualName)
}

func (db *darwinBackend) PeerStats(_ context.Context, name string, peerPublicKey string) (*backend.PeerStats, error) {
	db.mu.RLock()
	managed, exists := db.devices[name]
	db.mu.RUnlock()

	actualName := name
	if exists {
		actualName = managed.name
	}

	device, err := db.client.Device(actualName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return peerStats(device, peerPublicKey)
}

func (db *darwinBackend) FindForeignServers(_ context.Context, knownInterfaces []string) ([]*backend.ForeignServer, error) {
	devices, err := db.client.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	var foreignServers []*backend.ForeignServer
	for _, device := range devices {
		// Normalize known interface names for comparison
		normalizedKnown := make([]string, len(knownInterfaces))
		for i, iface := range knownInterfaces {
			normalizedKnown[i] = normalizeInterfaceName(iface)
		}

		if slices.Contains(normalizedKnown, device.Name) {
			continue
		}

		foreignServer, err := deviceToForeignServer(device)
		if err != nil {
			continue
		}

		foreignServers = append(foreignServers, foreignServer)
	}

	return foreignServers, nil
}

func (db *darwinBackend) configureWireguard(name string, opts backend.WireguardOptions) error {
	key, err := wgtypes.ParseKey(opts.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Get current device state
	currentDevice, err := db.client.Device(name)
	if err != nil {
		return fmt.Errorf("failed to get current device state: %w", err)
	}

	// Compute peers (add new, update existing, remove old)
	peers, err := computePeers(currentDevice, opts.Peers)
	if err != nil {
		return fmt.Errorf("failed to compute peers: %w", err)
	}

	// FirewallMark is not supported on macOS, set to nil
	cfg := wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   opts.ListenPort,
		FirewallMark: nil,
		ReplacePeers: false,
		Peers:        peers,
	}

	if err := db.client.ConfigureDevice(name, cfg); err != nil {
		return fmt.Errorf("failed to configure device: %w", err)
	}

	return nil
}

func (db *darwinBackend) Close(_ context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Close all managed devices
	for name, managed := range db.devices {
		if managed.uapiListener != nil {
			managed.uapiListener.Close()
		}
		managed.device.Close()
		delete(db.devices, name)
	}

	// Close wgctrl client
	return db.client.Close()
}

func (db *darwinBackend) Supported() bool {
	return true
}

// normalizeInterfaceName ensures interface names follow macOS utun format
func normalizeInterfaceName(name string) string {
	// If already in utun format, return as-is
	if strings.HasPrefix(name, "utun") {
		return name
	}

	// Extract numeric suffix if present (e.g., "wg0" -> "0")
	numStr := ""
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] >= '0' && name[i] <= '9' {
			numStr = string(name[i]) + numStr
		} else {
			break
		}
	}

	// If we found a number, use it
	if numStr != "" {
		return "utun" + numStr
	}

	// Otherwise, let kernel choose
	return "utun"
}

func wgDeviceToBackendDevice(device *wgtypes.Device, name string) (*backend.Device, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface addresses: %w", err)
	}

	return &backend.Device{
		Interface: backend.Interface{
			Name: iface.Name,
			Addresses: adapt.Array(addrs, func(addr net.Addr) string {
				return addr.String()
			}),
			Mtu: iface.MTU,
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

func computePeers(device *wgtypes.Device, peerOptions []*backend.PeerOptions) ([]wgtypes.PeerConfig, error) {
	var actualPeers []wgtypes.PeerConfig
	for _, p := range peerOptions {
		peerConfig, err := peerOptionsToPeerConfig(p)
		if err != nil {
			return nil, err
		}
		actualPeers = append(actualPeers, peerConfig)
	}

	var peers []wgtypes.PeerConfig

	// Update or remove existing peers
	for _, currentPeer := range device.Peers {
		var found bool
		for _, actualPeer := range actualPeers {
			if currentPeer.PublicKey == actualPeer.PublicKey {
				found = true
				actualPeer.UpdateOnly = true
				peers = append(peers, actualPeer)
				break
			}
		}
		if !found {
			// Remove peer
			peers = append(peers, wgtypes.PeerConfig{
				PublicKey: currentPeer.PublicKey,
				Remove:    true,
			})
		}
	}

	// Add new peers
	for _, actualPeer := range actualPeers {
		var found bool
		for _, currentPeer := range device.Peers {
			if actualPeer.PublicKey == currentPeer.PublicKey {
				found = true
				break
			}
		}
		if !found {
			peers = append(peers, actualPeer)
		}
	}

	return peers, nil
}

func peerOptionsToPeerConfig(opts *backend.PeerOptions) (wgtypes.PeerConfig, error) {
	publicKey, err := wgtypes.ParseKey(opts.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("invalid public key: %w", err)
	}

	var presharedKey *wgtypes.Key
	if opts.PresharedKey != "" {
		key, err := wgtypes.ParseKey(opts.PresharedKey)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid preshared key: %w", err)
		}
		presharedKey = &key
	}

	var endpoint *net.UDPAddr
	if opts.Endpoint != "" {
		addr, err := net.ResolveUDPAddr("udp", opts.Endpoint)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid endpoint: %w", err)
		}
		endpoint = addr
	}

	// Parse allowed IPs from CIDR strings to net.IPNet
	allowedIPs := make([]net.IPNet, 0, len(opts.AllowedIPs))
	for _, cidr := range opts.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid allowed IP %s: %w", cidr, err)
		}
		allowedIPs = append(allowedIPs, *ipNet)
	}

	// Convert persistent keepalive from int (seconds) to time.Duration
	var persistentKeepaliveInterval *time.Duration
	if opts.PersistentKeepalive != 0 {
		interval := time.Duration(opts.PersistentKeepalive) * time.Second
		persistentKeepaliveInterval = &interval
	}

	return wgtypes.PeerConfig{
		PublicKey:                   publicKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                presharedKey,
		Endpoint:                    endpoint,
		PersistentKeepaliveInterval: persistentKeepaliveInterval,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  allowedIPs,
	}, nil
}

func peerStats(device *wgtypes.Device, peerPublicKey string) (*backend.PeerStats, error) {
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

func deviceToForeignServer(device *wgtypes.Device) (*backend.ForeignServer, error) {
	iface, err := net.InterfaceByName(device.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface addresses: %w", err)
	}

	state := "down"
	if iface.Flags&net.FlagUp != 0 {
		state = "up"
	}

	return &backend.ForeignServer{
		Interface: &backend.ForeignInterface{
			Name: iface.Name,
			Addresses: adapt.Array(addrs, func(addr net.Addr) string {
				return addr.String()
			}),
			Mtu:   iface.MTU,
			State: state,
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
