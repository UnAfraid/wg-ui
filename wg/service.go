package wg

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	updateInterval = time.Minute
)

type Service interface {
	Close() error
	ForeignServers(ctx context.Context) (foreignServers []*ForeignServer, err error)
	ImportForeignServer(ctx context.Context, name string, userId string) (*server.Server, error)
	StartServer(ctx context.Context, serverId string) (*server.Server, error)
	StopServer(ctx context.Context, serverUd string) (*server.Server, error)
	ConfigureWireGuard(name string, privateKey string, listenPort *int, firewallMark *int, peers []*peer.Peer) error
	ConfigureInterface(name string, address string, mtu int) error
	DeleteInterface(name string) error
	InterfaceStats(name string) (*InterfaceStats, error)
	PeerStats(name string, peerPublicKey string) (*PeerStats, error)
	AddPeer(name string, privateKey string, listenPort *int, firewallMark *int, peer *peer.Peer) error
	UpdatePeer(name string, privateKey string, listenPort *int, firewallMark *int, peer *peer.Peer) error
	RemovePeer(name string, privateKey string, listenPort *int, firewallMark *int, peer *peer.Peer) error
}

type service struct {
	client        *wgctrl.Client
	updateStop    func()
	serverService server.Service
	peerService   peer.Service
	stopChan      chan struct{}
	stoppedChan   chan struct{}
}

func NewService(serverService server.Service, peerService peer.Service) (Service, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	s := &service{
		client:        client,
		serverService: serverService,
		peerService:   peerService,
		stopChan:      make(chan struct{}),
		stoppedChan:   make(chan struct{}),
	}

	if err := s.init(); err != nil {
		return nil, err
	}

	go s.run()

	return s, nil
}

func (s *service) init() error {
	servers, err := s.serverService.FindServers(context.Background(), &server.FindOptions{})
	if err != nil {
		return fmt.Errorf("failed to find servers: %w", err)
	}

	for _, svc := range servers {
		if !svc.Enabled {
			continue
		}

		if _, err := s.StartServer(context.Background(), svc.Id); err != nil {
			logrus.WithError(err).Error("failed to start server")
			return nil
		}
	}

	return nil
}

func (s *service) run() {
	defer close(s.stoppedChan)
	for {
		select {
		case <-s.stopChan:
			return
		case <-time.After(updateInterval):
			s.update()
		}
	}
}

func (s *service) update() {
	servers, err := s.serverService.FindServers(context.Background(), &server.FindOptions{})
	if err != nil {
		logrus.
			WithError(err).
			Error("failed to find servers")
		return
	}

	for _, svc := range servers {
		if !svc.Enabled || !svc.Running {
			continue
		}

		wg, err := s.client.Device(svc.Name)
		if err != nil {
			logrus.
				WithError(err).
				WithField("serverId", svc.Id).
				WithField("serverName", svc.Name).
				Error("failed to find open wireguard device")
			return
		}

		if adapt.Dereference(svc.ListenPort) != wg.ListenPort {
			updateOptions := &server.UpdateOptions{ListenPort: &wg.ListenPort}
			updateFieldMask := &server.UpdateFieldMask{ListenPort: true}
			svc, err = s.serverService.UpdateServer(context.Background(), svc.Id, updateOptions, updateFieldMask, "")
			if err != nil {
				logrus.
					WithError(err).
					WithField("serverId", svc.Id).
					WithField("serverName", svc.Name).
					Error("failed to update wireguard server")
				return
			}
		}

		for _, p := range wg.Peers {
			existingPeer, err := s.peerService.FindPeer(context.Background(), &peer.FindOneOptions{
				ServerIdPublicKeyOption: &peer.ServerIdPublicKeyOption{
					ServerId:  svc.Id,
					PublicKey: p.PublicKey.String(),
				},
			})
			if err != nil {
				logrus.
					WithError(err).
					WithField("serverId", svc.Id).
					WithField("serverName", svc.Name).
					WithField("peerPublicKey", p.PublicKey.String()).
					Error("failed to to find peer")
				continue
			}
			if existingPeer == nil {
				continue
			}

			if p.Endpoint == nil || p.Endpoint.String() == existingPeer.Endpoint {
				continue
			}

			options := &peer.UpdateOptions{
				Endpoint: p.Endpoint.String(),
			}
			fieldMask := &peer.UpdateFieldMask{
				Endpoint: true,
			}

			_, err = s.peerService.UpdatePeer(context.Background(), existingPeer.Id, options, fieldMask, "")
			if err != nil {
				logrus.
					WithError(err).
					WithField("serverId", svc.Id).
					WithField("serverName", svc.Name).
					WithField("peerId", existingPeer.Id).
					WithField("peerPublicKey", p.PublicKey.String()).
					Error("failed to to update peer")
				return
			}
		}
	}
}

func (s *service) Close() error {
	close(s.stopChan)
	<-s.stoppedChan
	return s.client.Close()
}

func (s *service) ForeignServers(ctx context.Context) (foreignServers []*ForeignServer, err error) {
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find servers: %w", err)
	}

	knownInterfaces := adapt.Array(servers, func(server *server.Server) string {
		return server.Name
	})

	foreignInterfaces, err := findForeignInterfaces(knownInterfaces)
	if err != nil {
		return nil, fmt.Errorf("failed to find foreign interfaces: %w", err)
	}

	for i, foreignInterface := range foreignInterfaces {
		device, err := s.client.Device(foreignInterface.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to open wireguard interface: %s", foreignInterface.Name)
		}

		foreignServers = append(foreignServers, &ForeignServer{
			ForeignInterface: &foreignInterfaces[i],
			Name:             device.Name,
			Type:             device.Type.String(),
			PublicKey:        device.PublicKey.String(),
			ListenPort:       device.ListenPort,
			FirewallMark:     device.FirewallMark,
			Peers: adapt.Array(device.Peers, func(peer wgtypes.Peer) *ForeignPeer {
				var endpoint *string
				if peer.Endpoint != nil {
					endpoint = adapt.ToPointer(peer.Endpoint.String())
				}
				return &ForeignPeer{
					PublicKey:                   peer.PublicKey.String(),
					Endpoint:                    endpoint,
					PersistentKeepaliveInterval: peer.PersistentKeepaliveInterval.Seconds(),
					LastHandshakeTime:           peer.LastHandshakeTime,
					ReceiveBytes:                peer.ReceiveBytes,
					TransmitBytes:               peer.TransmitBytes,
					AllowedIPs: adapt.Array(peer.AllowedIPs, func(allowedIp net.IPNet) string {
						return allowedIp.String()
					}),
					ProtocolVersion: peer.ProtocolVersion,
				}
			}),
		})
	}
	return foreignServers, nil
}

func (s *service) ImportForeignServer(ctx context.Context, name string, userId string) (*server.Server, error) {
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find servers: %w", err)
	}

	knownInterfaces := adapt.Array(servers, func(server *server.Server) string {
		return server.Name
	})

	foreignInterfaces, err := findForeignInterfaces(knownInterfaces)
	if err != nil {
		return nil, fmt.Errorf("failed to find foreign interfaces: %w", err)
	}

	var foreignInterface *ForeignInterface
	for _, fn := range foreignInterfaces {
		if strings.EqualFold(fn.Name, name) {
			foreignInterface = &fn
			break
		}
	}

	if foreignInterface == nil {
		return nil, fmt.Errorf("foreign interface: %s not found", name)
	}

	device, err := s.client.Device(foreignInterface.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface: %s", foreignInterface.Name)
	}

	var address string
	if len(foreignInterface.Addresses) != 0 {
		address = foreignInterface.Addresses[0]
	}

	createServer, err := s.serverService.CreateServer(ctx, &server.CreateOptions{
		Name:         foreignInterface.Name,
		Description:  "",
		Enabled:      true,
		PublicKey:    device.PublicKey.String(),
		PrivateKey:   device.PrivateKey.String(),
		ListenPort:   adapt.ToPointerNilZero(device.ListenPort),
		FirewallMark: adapt.ToPointerNilZero(device.FirewallMark),
		Address:      address,
		DNS:          nil,
		MTU:          foreignInterface.Mtu,
	}, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	for i, p := range device.Peers {
		var endpoint string
		if p.Endpoint != nil {
			endpoint = p.Endpoint.String()
		}

		_, err := s.peerService.CreatePeer(ctx, createServer.Id, &peer.CreateOptions{
			Name:        fmt.Sprintf("Peer #%d", i+1),
			Description: "",
			PublicKey:   p.PublicKey.String(),
			Endpoint:    endpoint,
			AllowedIPs: adapt.Array(p.AllowedIPs, func(allowedIp net.IPNet) string {
				return allowedIp.String()
			}),
			PresharedKey:        p.PresharedKey.String(),
			PersistentKeepalive: int(p.PersistentKeepaliveInterval.Seconds()),
		}, userId)
		if err != nil {
			return nil, fmt.Errorf("failed to create peer: %w", err)
		}
	}

	return createServer, nil
}

func (s *service) StartServer(ctx context.Context, serverId string) (*server.Server, error) {
	svc, err := s.serverService.FindServer(ctx, &server.FindOneOptions{
		IdOption: &server.IdOption{
			Id: serverId,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find server: %w", err)
	}
	if svc == nil {
		return nil, server.ErrServerNotFound
	}
	if !svc.Enabled {
		return svc, nil
	}

	peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
		ServerId: &svc.Id,
	})
	if err != nil {
		return nil, err
	}

	logrus.
		WithField("name", svc.Name).
		Info("starting wireguard")

	if err := s.ConfigureInterface(svc.Name, svc.Address, svc.MTU); err != nil {
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}

	if err := s.ConfigureWireGuard(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peers); err != nil {
		return nil, fmt.Errorf("failed to configure wireguard: %w", err)
	}

	currentDevice, err := s.client.Device(svc.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to open wireguard device: %w", err)
	}

	updateServerOptions := &server.UpdateOptions{
		ListenPort: &currentDevice.ListenPort,
		Running:    true,
	}
	updateServerFieldMask := &server.UpdateFieldMask{
		ListenPort: true,
		Running:    true,
	}
	return s.serverService.UpdateServer(ctx, serverId, updateServerOptions, updateServerFieldMask, "")
}

func (s *service) StopServer(ctx context.Context, serverId string) (*server.Server, error) {
	svc, err := s.serverService.FindServer(ctx, &server.FindOneOptions{
		IdOption: &server.IdOption{
			Id: serverId,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find server: %w", err)
	}
	if svc == nil {
		return nil, server.ErrServerNotFound
	}
	if !svc.Enabled {
		return svc, nil
	}

	logrus.
		WithField("name", svc.Name).
		Info("stopping wireguard")

	if err := s.DeleteInterface(svc.Name); err != nil {
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}

	updateServerOptions := &server.UpdateOptions{
		Running: false,
	}
	updateServerFieldMask := &server.UpdateFieldMask{
		Running: true,
	}
	return s.serverService.UpdateServer(ctx, serverId, updateServerOptions, updateServerFieldMask, "")
}

func (s *service) ConfigureWireGuard(name string, privateKey string, listenPort *int, firewallMark *int, peers []*peer.Peer) error {
	currentDevice, err := s.client.Device(name)
	if err != nil {
		return fmt.Errorf("failed to open wireguard device: %w", err)
	}

	var actualPeers []wgtypes.PeerConfig
	for _, p := range peers {
		peerConfig, err := toPeerConfig(p)
		if err != nil {
			return err
		}
		actualPeers = append(actualPeers, peerConfig)
	}

	var differentPeers []wgtypes.PeerConfig
	for _, currentPeer := range currentDevice.Peers {
		var found bool
		for _, actualPeer := range actualPeers {
			if currentPeer.PublicKey == actualPeer.PublicKey {
				found = true
				actualPeer.UpdateOnly = true
				differentPeers = append(differentPeers, actualPeer)
				break
			}
		}
		if !found {
			peerToRemove := wgtypes.PeerConfig{
				PublicKey: currentPeer.PublicKey,
				Remove:    true,
			}
			differentPeers = append(differentPeers, peerToRemove)
		}
	}

	for _, actualPeer := range actualPeers {
		var found bool
		for _, currentPeer := range currentDevice.Peers {
			if actualPeer.PublicKey == currentPeer.PublicKey {
				found = true
				break
			}
		}
		if !found {
			differentPeers = append(differentPeers, actualPeer)
		}
	}

	return s.configureWireguard(name, privateKey, listenPort, firewallMark, differentPeers...)
}

func (s *service) ConfigureInterface(name string, address string, mtu int) error {
	return configureInterface(name, address, mtu)
}

func (s *service) DeleteInterface(name string) error {
	return deleteInterface(name)
}

func (s *service) InterfaceStats(name string) (*InterfaceStats, error) {
	return interfaceStats(name)
}

func (s *service) PeerStats(name string, peerPublicKey string) (*PeerStats, error) {
	publicKey, err := wgtypes.ParseKey(peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid peer: %s public key: %w", name, err)
	}

	currentDevice, err := s.client.Device(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open wireguard device: %w", err)
	}

	for _, p := range currentDevice.Peers {
		if p.PublicKey == publicKey {
			return &PeerStats{
				LastHandshakeTime: p.LastHandshakeTime,
				ReceiveBytes:      p.ReceiveBytes,
				TransmitBytes:     p.TransmitBytes,
				ProtocolVersion:   p.ProtocolVersion,
			}, nil
		}
	}

	return nil, nil
}

func (s *service) AddPeer(name string, privateKey string, listenPort *int, firewallMark *int, p *peer.Peer) error {
	currentDevice, err := s.client.Device(name)
	if err != nil {
		return fmt.Errorf("failed to open wireguard device: %w", err)
	}

	peerConfig, err := toPeerConfig(p)
	if err != nil {
		return err
	}

	var currentPeer *wgtypes.Peer
	for _, p := range currentDevice.Peers {
		if p.PublicKey == peerConfig.PublicKey {
			currentPeer = &p
			break
		}
	}

	if currentPeer != nil {
		peerConfig.UpdateOnly = true
		if len(currentPeer.AllowedIPs) != len(peerConfig.AllowedIPs) {
			peerConfig.ReplaceAllowedIPs = true
		} else {
			for i := 0; i < len(currentPeer.AllowedIPs); i++ {
				if currentPeer.AllowedIPs[i].String() != peerConfig.AllowedIPs[i].String() {
					peerConfig.ReplaceAllowedIPs = true
					break
				}
			}
		}
	}

	return s.configureWireguard(name, privateKey, listenPort, firewallMark, peerConfig)
}

func (s *service) UpdatePeer(name string, privateKey string, listenPort *int, firewallMark *int, p *peer.Peer) error {
	currentDevice, err := s.client.Device(name)
	if err != nil {
		return fmt.Errorf("failed to open wireguard device: %w", err)
	}

	peerConfig, err := toPeerConfig(p)
	if err != nil {
		return err
	}
	peerConfig.UpdateOnly = true

	var currentPeer *wgtypes.Peer
	for _, p := range currentDevice.Peers {
		if p.PublicKey == peerConfig.PublicKey {
			currentPeer = &p
			break
		}
	}
	if currentPeer == nil {
		return nil
	}

	peerConfig.UpdateOnly = true
	if len(currentPeer.AllowedIPs) != len(peerConfig.AllowedIPs) {
		peerConfig.ReplaceAllowedIPs = true
	} else {
		for i := 0; i < len(currentPeer.AllowedIPs); i++ {
			if currentPeer.AllowedIPs[i].String() != peerConfig.AllowedIPs[i].String() {
				peerConfig.ReplaceAllowedIPs = true
				break
			}
		}
	}

	return s.configureWireguard(name, privateKey, listenPort, firewallMark, peerConfig)
}

func (s *service) RemovePeer(name string, privateKey string, listenPort *int, firewallMark *int, p *peer.Peer) error {
	currentDevice, err := s.client.Device(name)
	if err != nil {
		return fmt.Errorf("failed to open wireguard device: %w", err)
	}

	peerConfig, err := toPeerConfig(p)
	if err != nil {
		return err
	}

	var currentPeer *wgtypes.Peer
	for _, p := range currentDevice.Peers {
		if p.PublicKey == peerConfig.PublicKey {
			currentPeer = &p
			break
		}
	}
	if currentPeer == nil {
		return nil
	}
	peerConfig.Remove = true

	return s.configureWireguard(name, privateKey, listenPort, firewallMark, peerConfig)
}

func (s *service) configureWireguard(name string, privateKey string, listenPort *int, firewallMark *int, peers ...wgtypes.PeerConfig) error {
	key, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return fmt.Errorf("invalid server private key: %w", err)
	}

	wgConfig := wgtypes.Config{
		PrivateKey:   &key,
		ListenPort:   listenPort,
		FirewallMark: firewallMark,
		ReplacePeers: false,
		Peers:        peers,
	}

	if err = s.client.ConfigureDevice(name, wgConfig); err != nil {
		return fmt.Errorf("failed to configure device: %w", err)
	}

	return nil
}

func toPeerConfig(peer *peer.Peer) (wgtypes.PeerConfig, error) {
	publicKey, err := wgtypes.ParseKey(peer.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("invalid peer: %s public key: %w", peer.Name, err)
	}

	var presharedKey *wgtypes.Key
	if peer.PresharedKey != "" {
		key, err := wgtypes.ParseKey(peer.PresharedKey)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid peer: %s preshared key - %w", peer.Name, err)
		}
		presharedKey = &key
	}

	allowedIPs := make([]net.IPNet, len(peer.AllowedIPs))
	for i, cidr := range peer.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
		allowedIPs[i] = *ipNet
	}

	endpointAddrPort, err := netip.ParseAddrPort(peer.Endpoint)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("invalid peer: %s endpoint - %w", peer.Name, err)
	}

	var persistentKeepaliveInterval *time.Duration
	if peer.PersistentKeepalive != 0 {
		persistentKeepaliveInterval = adapt.ToPointer(time.Duration(peer.PersistentKeepalive) * time.Second)
	}

	return wgtypes.PeerConfig{
		PublicKey:                   publicKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                presharedKey,
		Endpoint:                    net.UDPAddrFromAddrPort(endpointAddrPort),
		PersistentKeepaliveInterval: persistentKeepaliveInterval,
		ReplaceAllowedIPs:           false,
		AllowedIPs:                  allowedIPs,
	}, nil
}
