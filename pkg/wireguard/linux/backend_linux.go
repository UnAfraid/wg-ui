//go:build linux

package linux

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func Register() {
	driver.Register("linux", func(_ context.Context, rawURL string) (driver.Backend, error) {
		return NewLinuxBackend(rawURL)
	}, true)
}

type linuxBackend struct {
	client *wgctrl.Client
}

func NewLinuxBackend(_ string) (driver.Backend, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize linux backend: %w", err)
	}

	return &linuxBackend{
		client: client,
	}, nil
}

func (lb *linuxBackend) Device(_ context.Context, name string) (*driver.Device, error) {
	device, err := lb.client.Device(name)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %s", err)
	}

	return wgDeviceToBackendDevice(device, name)
}

func (lb *linuxBackend) Up(_ context.Context, options driver.ConfigureOptions) (*driver.Device, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	interfaceOptions := options.InterfaceOptions
	if err := configureInterface(interfaceOptions.Name, interfaceOptions.Address, interfaceOptions.Mtu); err != nil {
		return nil, fmt.Errorf("failed to configure interface: %s - %w", interfaceOptions.Name, err)
	}

	wireguardOptions := options.WireguardOptions
	if err := lb.configureWireguard(interfaceOptions.Name, wireguardOptions.PrivateKey, wireguardOptions.ListenPort, wireguardOptions.FirewallMark, wireguardOptions.Peers); err != nil {
		return nil, fmt.Errorf("failed to configure wireguard: %s - %w", interfaceOptions.Name, err)
	}

	device, err := lb.client.Device(interfaceOptions.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to find device: %s", err)
	}

	return wgDeviceToBackendDevice(device, interfaceOptions.Name)
}

func wgDeviceToBackendDevice(device *wgtypes.Device, name string) (*driver.Device, error) {
	link, err := findInterface(name)
	if err != nil {
		return nil, err
	}

	addressList, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %s address list: %w", name, err)
	}

	return &driver.Device{
		Interface: driver.Interface{
			Name: link.Attrs().Name,
			Addresses: adapt.Array(addressList, func(addr netlink.Addr) string {
				return addr.String()
			}),
			Mtu: link.Attrs().MTU,
		},
		Wireguard: driver.Wireguard{
			Name:         device.Name,
			PublicKey:    device.PublicKey.String(),
			PrivateKey:   device.PrivateKey.String(),
			ListenPort:   device.ListenPort,
			FirewallMark: device.FirewallMark,
			Peers: adapt.Array(device.Peers, func(peer wgtypes.Peer) *driver.Peer {
				var endpoint string
				if peer.Endpoint != nil {
					endpoint = peer.Endpoint.String()
				}
				return &driver.Peer{
					PublicKey:           peer.PublicKey.String(),
					Endpoint:            endpoint,
					AllowedIPs:          peer.AllowedIPs,
					PresharedKey:        peer.PresharedKey.String(),
					PersistentKeepalive: peer.PersistentKeepaliveInterval,
					Stats: driver.PeerStats{
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

func (lb *linuxBackend) Down(_ context.Context, name string) error {
	return deleteInterface(name)
}

func (lb *linuxBackend) Status(_ context.Context, name string) (bool, error) {
	link, err := findInterface(name)
	if err != nil {
		return false, err
	}
	return link != nil, nil
}

func (lb *linuxBackend) Stats(_ context.Context, name string) (*driver.InterfaceStats, error) {
	return interfaceStats(name)
}

func (lb *linuxBackend) PeerStats(_ context.Context, name string, peerPublicKey string) (*driver.PeerStats, error) {
	currentDevice, err := lb.client.Device(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open wireguard device: %w", err)
	}
	return peerStats(currentDevice, name, peerPublicKey)
}

func (lb *linuxBackend) FindForeignServers(_ context.Context, knownInterfaces []string) ([]*driver.ForeignServer, error) {
	return lb.findForeignServers(knownInterfaces)
}

func (lb *linuxBackend) configureWireguard(name string, privateKey string, listenPort *int, firewallMark *int, peerOptions []*driver.PeerOptions) error {
	device, err := lb.client.Device(name)
	if err != nil {
		return fmt.Errorf("failed to open wireguard device: %w", err)
	}

	key, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	peers, err := computePeers(device, peerOptions)
	if err != nil {
		return fmt.Errorf("failed to compute peers: %w", err)
	}

	return lb.applyDeviceConfiguration(device, name, &key, listenPort, firewallMark, peers)
}

func (lb *linuxBackend) Close(_ context.Context) error {
	return lb.client.Close()
}

func computePeers(device *wgtypes.Device, peerOptions []*driver.PeerOptions) ([]wgtypes.PeerConfig, error) {
	var actualPeers []wgtypes.PeerConfig
	for _, p := range peerOptions {
		peerConfig, err := wireguardPeerOptionsToPeerConfig(p)
		if err != nil {
			return nil, err
		}
		actualPeers = append(actualPeers, peerConfig)
	}

	var peers []wgtypes.PeerConfig
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
			peerToRemove := wgtypes.PeerConfig{
				PublicKey: currentPeer.PublicKey,
				Remove:    true,
			}
			peers = append(peers, peerToRemove)
		}
	}

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

func (lb *linuxBackend) applyDeviceConfiguration(
	device *wgtypes.Device,
	name string,
	privateKey *wgtypes.Key,
	listenPort *int,
	firewallMark *int,
	peers []wgtypes.PeerConfig,
) error {
	wgConfig := wgtypes.Config{
		PrivateKey:   privateKey,
		ListenPort:   listenPort,
		FirewallMark: firewallMark,
		ReplacePeers: false,
		Peers:        peers,
	}

	if err := lb.client.ConfigureDevice(name, wgConfig); err != nil {
		return fmt.Errorf("failed to configure device: %w", err)
	}

	var allowedIPs []net.IPNet
	for _, p := range device.Peers {
		allowedIPs = append(allowedIPs, p.AllowedIPs...)
	}

	if err := configureRoutes(name, allowedIPs); err != nil {
		return fmt.Errorf("failed to configure routes: %w", err)
	}

	return nil
}

func findInterface(name string) (netlink.Link, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if os.IsNotExist(err) || errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find link by name: %w", err)
	}
	return link, nil
}

func configureInterface(name string, address string, mtu int) error {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name

	link := &wgLink{
		attrs: &attrs,
	}

	if err := netlink.LinkAdd(link); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to add interface: %w", err)
		}
	}

	addressList, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to get interface: %s address list: %w", name, err)
	}

	serverAddress, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("failed to parse interface address: %w", err)
	}

	needsAddress := true
	for _, addr := range addressList {
		if addr.Equal(*serverAddress) {
			needsAddress = false
			break
		}
	}

	if needsAddress {
		if err = netlink.AddrAdd(link, serverAddress); err != nil {
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add address: %w", err)
			}
		}
	}

	if mtu != attrs.MTU {
		if err = netlink.LinkSetMTU(link, mtu); err != nil {
			return fmt.Errorf("failed to set server mtu: %w", err)
		}
	}

	if attrs.OperState != netlink.OperUp {
		if err = netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to set interface up: %w", err)
		}
	}

	return nil
}

func deleteInterface(name string) error {
	link, err := findInterface(name)
	if err != nil {
		return err
	}
	if link == nil {
		return nil
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface down: %w", err)
	}
	return nil
}

func interfaceStats(name string) (*driver.InterfaceStats, error) {
	link, err := findInterface(name)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, nil
	}
	return linkStatisticsToBackendInterfaceStats(link.Attrs().Statistics), nil
}

func peerStats(device *wgtypes.Device, name string, peerPublicKey string) (*driver.PeerStats, error) {
	publicKey, err := wgtypes.ParseKey(peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid peer: %s public key: %w", name, err)
	}

	for _, p := range device.Peers {
		if p.PublicKey == publicKey {
			return &driver.PeerStats{
				LastHandshakeTime: p.LastHandshakeTime,
				ReceiveBytes:      p.ReceiveBytes,
				TransmitBytes:     p.TransmitBytes,
				ProtocolVersion:   p.ProtocolVersion,
			}, nil
		}
	}

	return nil, nil
}

func configureRoutes(name string, allowedIPs []net.IPNet) error {
	link, err := findInterface(name)
	if err != nil {
		return fmt.Errorf("failed to find link by name: %w", err)
	}
	if link == nil {
		return fmt.Errorf("interface not found: %s", name)
	}

	routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to get routes: %w", err)
	}

	routesToAdd, routesToUpdate, routesToRemove := computeRoutes(link, routes, allowedIPs)

	for i, route := range routesToAdd {
		if err = netlink.RouteAdd(routesToAdd[i]); err != nil {
			return fmt.Errorf("failed to add route for %s - %w", route.Dst.String(), err)
		}

		logrus.
			WithField("name", link.Attrs().Name).
			WithField("route", route.Dst.String()).
			Debug("route added")
	}

	for i, route := range routesToUpdate {
		if err = netlink.RouteReplace(routesToUpdate[i]); err != nil {
			return fmt.Errorf("failed to replace route for %s - %w", route.Dst.String(), err)
		}

		logrus.
			WithField("name", link.Attrs().Name).
			WithField("route", route.Dst.String()).
			Debug("route replaced")
	}

	for i, route := range routesToRemove {
		if err = netlink.RouteDel(routesToRemove[i]); err != nil {
			return fmt.Errorf("failed to delete route for %s - %w", route.Dst.String(), err)
		}

		logrus.
			WithField("name", link.Attrs().Name).
			WithField("route", route.Dst.String()).
			Debug("route deleted")
	}
	return nil
}

func computeRoutes(link netlink.Link, existingRoutes []netlink.Route, allowedIPs []net.IPNet) ([]*netlink.Route, []*netlink.Route, []*netlink.Route) {
	var routesToAdd []*netlink.Route
	var routesToUpdate []*netlink.Route
	var routesToRemove []*netlink.Route
	for i, allowedIP := range allowedIPs {
		var existingRoute *netlink.Route
		for _, route := range existingRoutes {
			if route.Dst != nil && route.Dst.IP.Equal(allowedIP.IP) && slices.Equal(route.Dst.Mask, allowedIP.Mask) {
				existingRoute = &existingRoutes[i]
				break
			}
		}
		if existingRoute != nil {
			var update bool
			if existingRoute.Scope != netlink.SCOPE_LINK {
				existingRoute.Scope = netlink.SCOPE_LINK
				update = true
			}

			if existingRoute.Protocol != netlink.RouteProtocol(3) {
				existingRoute.Protocol = netlink.RouteProtocol(3)
				update = true
			}

			if existingRoute.Type != 1 {
				existingRoute.Type = 1
				update = true
			}

			if update {
				routesToUpdate = append(routesToUpdate, existingRoute)
			}
			continue
		}

		routesToAdd = append(routesToAdd, &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       &allowedIPs[i],
			Protocol:  netlink.RouteProtocol(3),
			Type:      1,
		})
	}

	for i, existingRoute := range existingRoutes {
		var exists bool
		for _, allowedIP := range allowedIPs {
			exists = existingRoute.Dst != nil && existingRoute.Dst.IP.Equal(allowedIP.IP) && slices.Equal(existingRoute.Dst.Mask, allowedIP.Mask)
			if exists {
				break
			}
		}
		if !exists {
			routesToRemove = append(routesToRemove, &existingRoutes[i])
		}
	}

	return routesToAdd, routesToUpdate, routesToRemove
}

func (lb *linuxBackend) findForeignServers(knownInterfaces []string) ([]*driver.ForeignServer, error) {
	list, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	var foreignServers []*driver.ForeignServer
	for _, link := range list {
		if !strings.EqualFold(link.Type(), "wireguard") {
			continue
		}

		if slices.Contains(knownInterfaces, link.Attrs().Name) {
			continue
		}

		foreignInterface, err := netlinkInterfaceToForeignInterface(link)
		if err != nil {
			return nil, err
		}

		device, err := lb.client.Device(foreignInterface.Name)
		if err != nil {
			return nil, err
		}

		foreignServers = append(foreignServers, &driver.ForeignServer{
			Interface:    foreignInterface,
			Name:         device.Name,
			Type:         device.Type.String(),
			PublicKey:    device.PublicKey.String(),
			ListenPort:   device.ListenPort,
			FirewallMark: device.FirewallMark,
			Peers: adapt.Array(device.Peers, func(peer wgtypes.Peer) *driver.Peer {
				var endpoint string
				if peer.Endpoint != nil {
					endpoint = peer.Endpoint.String()
				}
				return &driver.Peer{
					PublicKey:           peer.PublicKey.String(),
					Endpoint:            endpoint,
					AllowedIPs:          peer.AllowedIPs,
					PresharedKey:        peer.PresharedKey.String(),
					PersistentKeepalive: peer.PersistentKeepaliveInterval,
					Stats: driver.PeerStats{
						LastHandshakeTime: peer.LastHandshakeTime,
						ReceiveBytes:      peer.ReceiveBytes,
						TransmitBytes:     peer.TransmitBytes,
						ProtocolVersion:   peer.ProtocolVersion,
					},
				}
			}),
		})
	}
	return foreignServers, nil
}
