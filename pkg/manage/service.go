package manage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/dbx"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
	"github.com/UnAfraid/wg-ui/pkg/wireguard"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

type Service interface {
	Authenticate(ctx context.Context, username string, password string) (*user.User, error)
	CreateUser(ctx context.Context, options *user.CreateOptions) (*user.User, error)
	UpdateUser(ctx context.Context, userId string, options *user.UpdateOptions, fieldMask *user.UpdateFieldMask) (*user.User, error)
	DeleteUser(ctx context.Context, userId string) (*user.User, error)
	CreateBackend(ctx context.Context, options *backend.CreateOptions, userId string) (*backend.Backend, error)
	UpdateBackend(ctx context.Context, backendId string, options *backend.UpdateOptions, fieldMask *backend.UpdateFieldMask, userId string) (*backend.Backend, error)
	CreateServer(ctx context.Context, options *server.CreateOptions, userId string) (*server.Server, error)
	UpdateServer(ctx context.Context, serverId string, options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, userId string) (*server.Server, error)
	DeleteServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	StartServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	StopServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	ImportForeignServer(ctx context.Context, backendId string, name string, userId string) (*server.Server, error)
	CreatePeer(ctx context.Context, serverId string, options *peer.CreateOptions, userId string) (*peer.Peer, error)
	UpdatePeer(ctx context.Context, peerId string, options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask, userId string) (*peer.Peer, error)
	DeletePeer(ctx context.Context, peerId string, userId string) (*peer.Peer, error)
	PeerStats(ctx context.Context, serverId string, peerPublicKey string) (*driver.PeerStats, error)
	ForeignServers(ctx context.Context, backendId string) ([]*driver.ForeignServer, error)
	ForeignServersAll(ctx context.Context) ([]*driver.ForeignServer, error)
	DeleteBackend(ctx context.Context, backendId string, userId string) (*backend.Backend, error)
	Close()
}

type service struct {
	transactionScoper dbx.TransactionScoper
	userService       user.Service
	backendService    backend.Service
	serverService     server.Service
	peerService       peer.Service
	wireguardService  wireguard.Service
	stopChan          chan struct{}
	stoppedChan       chan struct{}
}

func NewService(
	transactionScoper dbx.TransactionScoper,
	userService user.Service,
	backendService backend.Service,
	serverService server.Service,
	peerService peer.Service,
	wireguardService wireguard.Service,
	automaticStatsUpdateInterval time.Duration,
	automaticStatsUpdateOnlyWithSubscribers bool,
) Service {
	s := &service{
		transactionScoper: transactionScoper,
		userService:       userService,
		backendService:    backendService,
		serverService:     serverService,
		peerService:       peerService,
		wireguardService:  wireguardService,
		stopChan:          make(chan struct{}),
		stoppedChan:       make(chan struct{}),
	}

	s.cleanup(context.Background())
	s.init()

	if automaticStatsUpdateInterval.Seconds() > 0 {
		go s.run(automaticStatsUpdateInterval, automaticStatsUpdateOnlyWithSubscribers)
	}

	return s
}

func (s *service) init() {
	ctx := context.Background()
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{
		Enabled: adapt.ToPointerNilZero(true),
	})
	if err != nil {
		logrus.WithError(err).Error("failed to find servers")
		return
	}

	// Find or create default linux backend for legacy servers
	var defaultBackend *backend.Backend
	for _, srv := range servers {
		if srv.BackendId == "" {
			if defaultBackend == nil {
				defaultBackend, err = s.getOrCreateDefaultBackend(ctx)
				if err != nil {
					logrus.WithError(err).Error("failed to get or create default backend for legacy servers")
					return
				}
			}
			// Update server to use default backend
			logrus.WithField("name", srv.Name).WithField("backend", defaultBackend.Name).Info("migrating legacy server to default backend")
			if _, err := s.serverService.UpdateServer(ctx, srv.Id, &server.UpdateOptions{
				BackendId: defaultBackend.Id,
			}, &server.UpdateFieldMask{
				BackendId: true,
			}, ""); err != nil {
				logrus.WithError(err).WithField("name", srv.Name).Error("failed to migrate legacy server to default backend")
				continue
			}
			srv.BackendId = defaultBackend.Id
		}
	}

	var initialized, failed int
	for _, srv := range servers {
		// Check if backend exists and is enabled
		b, err := s.findBackend(ctx, srv.BackendId)
		if err != nil {
			logrus.WithError(err).WithField("name", srv.Name).Warn("failed to find backend for server, skipping initialization")
			failed++
			continue
		}

		if !b.Enabled {
			logrus.WithField("name", srv.Name).WithField("backend", b.Name).Debug("backend is disabled, skipping server initialization")
			continue
		}

		peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
			ServerId: adapt.ToPointer(srv.Id),
		})
		if err != nil {
			logrus.WithError(err).WithField("name", srv.Name).Error("failed to find peers for server")
			failed++
			continue
		}

		if _, err = s.configureDevice(ctx, srv, peers); err != nil {
			logrus.WithError(err).WithField("name", srv.Name).Error("failed to configure wireguard device")
			failed++
			continue
		}

		initialized++
	}

	if failed > 0 {
		if initialized == 0 {
			logrus.WithField("failed", failed).Error("server initialization failed: no servers could be configured")
		} else {
			logrus.WithField("initialized", initialized).WithField("failed", failed).Warn("server initialization completed with errors")
		}
	} else if initialized > 0 {
		logrus.WithField("initialized", initialized).Info("server initialization completed")
	}
}

func (s *service) getOrCreateDefaultBackend(ctx context.Context) (*backend.Backend, error) {
	// Try to find existing linux backend
	backends, err := s.backendService.FindBackends(ctx, &backend.FindOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find backends: %w", err)
	}

	for _, b := range backends {
		if b.Type() == "linux" {
			return b, nil
		}
	}

	// Create default linux backend
	logrus.Info("creating default linux backend for legacy server migration")
	return s.backendService.CreateBackend(ctx, &backend.CreateOptions{
		Name:        "Linux (Default)",
		Description: "Default backend created for legacy server migration",
		Url:         "linux:///etc/wireguard",
		Enabled:     true,
	}, "")
}

func (s *service) run(interval time.Duration, automaticStatsUpdateOnlyWithSubscribers bool) {
	defer close(s.stoppedChan)
	ctx := context.Background()

	for {
		select {
		case <-s.stopChan:
			return
		case <-time.After(interval):
			if !automaticStatsUpdateOnlyWithSubscribers || s.serverService.HasSubscribers() {
				s.updateServersStats(ctx)
			}
		}
	}
}

func (s *service) Authenticate(ctx context.Context, username string, password string) (*user.User, error) {
	return s.userService.Authenticate(ctx, username, password)
}

func (s *service) CreateUser(ctx context.Context, options *user.CreateOptions) (*user.User, error) {
	return s.userService.CreateUser(ctx, options)
}

func (s *service) UpdateUser(ctx context.Context, userId string, options *user.UpdateOptions, fieldMask *user.UpdateFieldMask) (*user.User, error) {
	return s.userService.UpdateUser(ctx, userId, options, fieldMask)
}

func (s *service) DeleteUser(ctx context.Context, userId string) (*user.User, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*user.User, error) {
		deletedUser, err := s.userService.DeleteUser(ctx, userId)
		if err != nil {
			return nil, err
		}

		servers, err := s.serverService.FindServers(ctx, &server.FindOptions{
			CreateUserId: &deletedUser.Id,
			UpdateUserId: &deletedUser.Id,
		})
		if err != nil {
			logrus.WithError(err).Warn("failed to find servers")
		}
		for _, svc := range servers {
			s.cleanupOrphanedUserFromServer(ctx, svc)
		}

		peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
			CreateUserId: &deletedUser.Id,
			UpdateUserId: &deletedUser.Id,
		})
		if err != nil {
			logrus.WithError(err).Warn("failed to find peers")
		}
		for _, p := range peers {
			s.cleanupOrphanedUserFromPeer(ctx, p)
		}

		return deletedUser, nil
	})
}

func (s *service) CreateBackend(ctx context.Context, options *backend.CreateOptions, userId string) (*backend.Backend, error) {
	if options == nil {
		return nil, backend.ErrCreateBackendOptionsRequired
	}

	if err := s.testBackendURL(ctx, options.Url); err != nil {
		return nil, err
	}

	return s.backendService.CreateBackend(ctx, options, userId)
}

func (s *service) UpdateBackend(ctx context.Context, backendId string, options *backend.UpdateOptions, fieldMask *backend.UpdateFieldMask, userId string) (*backend.Backend, error) {
	if options == nil {
		return nil, backend.ErrUpdateBackendOptionsRequired
	}
	if fieldMask == nil {
		return nil, backend.ErrUpdateBackendFieldMaskRequired
	}

	if fieldMask.Url {
		existingBackend, err := s.findBackend(ctx, backendId)
		if err != nil {
			return nil, fmt.Errorf("failed to find backend: %w", err)
		}

		resolvedURL, err := backend.ReplaceRedactedURLPassword(options.Url, existingBackend.Url)
		if err != nil {
			return nil, err
		}
		options.Url = resolvedURL

		if err := s.testBackendURL(ctx, options.Url); err != nil {
			return nil, err
		}
	}

	return s.backendService.UpdateBackend(ctx, backendId, options, fieldMask, userId)
}

func (s *service) CreateServer(ctx context.Context, options *server.CreateOptions, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		createdServer, err := s.serverService.CreateServer(ctx, options, userId)
		if err != nil {
			return nil, err
		}

		if createdServer.Enabled {
			device, err := s.configureDevice(ctx, createdServer, nil)
			if err != nil {
				return nil, err
			}
			return s.updateServer(ctx, createdServer, device, userId)
		}

		return createdServer, nil
	})
}

func (s *service) UpdateServer(ctx context.Context, serverId string, options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		updatedServer, err := s.serverService.UpdateServer(ctx, serverId, options, fieldMask, userId)
		if err != nil {
			return nil, err
		}

		b, err := s.findBackend(ctx, updatedServer.BackendId)
		if err != nil {
			return nil, fmt.Errorf("failed to find backend: %w", err)
		}

		if !updatedServer.Enabled {
			status, err := s.wireguardService.Status(ctx, b, updatedServer.Name)
			if err != nil {
				return nil, err
			}

			if status {
				if err := s.wireguardService.Down(ctx, b, updatedServer.Name); err != nil {
					return nil, err
				}
				updateOptions := server.UpdateOptions{
					Running: false,
				}
				updateFieldMask := server.UpdateFieldMask{
					Running: true,
				}
				return s.serverService.UpdateServer(ctx, updatedServer.Id, &updateOptions, &updateFieldMask, userId)
			}
		}

		// Reconfigure live interfaces for updates that affect runtime/config state.
		if updatedServer.Running && serverUpdateRequiresReconfigure(fieldMask) {
			peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
				ServerId: &updatedServer.Id,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to find peers: %w", err)
			}

			if _, err := s.configureDevice(ctx, updatedServer, peers); err != nil {
				return nil, fmt.Errorf("failed to reconfigure device: %w", err)
			}
		}

		return updatedServer, nil
	})
}

func serverUpdateRequiresReconfigure(fieldMask *server.UpdateFieldMask) bool {
	if fieldMask == nil {
		return false
	}

	return fieldMask.Description ||
		fieldMask.PrivateKey ||
		fieldMask.ListenPort ||
		fieldMask.FirewallMark ||
		fieldMask.Address ||
		fieldMask.DNS ||
		fieldMask.MTU
}

func (s *service) DeleteServer(ctx context.Context, serverId string, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		svc, err := s.findServer(ctx, serverId)
		if err != nil {
			return nil, err
		}

		b, err := s.findBackend(ctx, svc.BackendId)
		if err != nil {
			logrus.WithError(err).WithField("backendId", svc.BackendId).Warn("failed to find backend")
		} else {
			if err = s.wireguardService.Down(ctx, b, svc.Name); err != nil {
				logrus.
					WithError(err).
					WithField("serverId", svc.Id).
					WithField("serverName", svc.Name).
					Warn("failed to stop server")
			}
		}

		deletedServer, err := s.serverService.DeleteServer(ctx, serverId, userId)
		if err != nil {
			return nil, err
		}

		peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
			ServerId: &svc.Id,
		})
		if err != nil {
			logrus.
				WithError(err).
				WithField("serverId", svc.Id).
				WithField("serverName", svc.Name).
				Warn("failed to find peers")
		}
		for _, p := range peers {
			s.cleanupOrphanedServerFromPeer(ctx, p)
		}

		return deletedServer, nil
	})
}

func (s *service) StartServer(ctx context.Context, serverId string, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		srv, err := s.findServer(ctx, serverId)
		if err != nil {
			return nil, err
		}

		peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
			ServerId: &srv.Id,
		})
		if err != nil {
			return nil, err
		}

		device, err := s.configureDevice(ctx, srv, peers)
		if err != nil {
			return nil, err
		}
		return s.updateServer(ctx, srv, device, userId)
	})
}

func (s *service) StopServer(ctx context.Context, serverId string, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		srv, err := s.findServer(ctx, serverId)
		if err != nil {
			return nil, err
		}

		b, err := s.findBackend(ctx, srv.BackendId)
		if err != nil {
			return nil, fmt.Errorf("failed to find backend: %w", err)
		}

		if err := s.wireguardService.Down(ctx, b, srv.Name); err != nil {
			return nil, err
		}

		updateOptions := server.UpdateOptions{
			Running: false,
		}
		updateFieldMask := server.UpdateFieldMask{
			Running: true,
		}
		return s.serverService.UpdateServer(ctx, srv.Id, &updateOptions, &updateFieldMask, userId)
	})
}

func (s *service) ImportForeignServer(ctx context.Context, backendId string, name string, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		b, err := s.findBackend(ctx, backendId)
		if err != nil {
			return nil, fmt.Errorf("failed to find backend: %w", err)
		}

		servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to find servers: %w", err)
		}
		managedServersByPublicKey := serverByPublicKey(servers, "")

		knownInterfaces := adapt.Array(servers, func(server *server.Server) string {
			return server.Name
		})

		foreignInterfaces, err := s.wireguardService.FindForeignServers(ctx, b, knownInterfaces)
		if err != nil {
			return nil, fmt.Errorf("failed to find foreign interfaces: %w", err)
		}

		var foreignServer *driver.ForeignServer
		for _, fn := range foreignInterfaces {
			if strings.EqualFold(fn.Name, name) {
				foreignServer = fn
				break
			}
		}

		if foreignServer == nil {
			return nil, fmt.Errorf("foreign interface: %s not found", name)
		}

		if existingServer := findServerByPublicKey(foreignServer.PublicKey, managedServersByPublicKey); existingServer != nil {
			return nil, s.serverPublicKeyConflictError(ctx, existingServer)
		}

		device, err := s.wireguardService.Device(ctx, b, foreignServer.Interface.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to open interface: %s", foreignServer.Interface.Name)
		}

		// Some backends may only discover the public key after reading the interface/config.
		if existingServer := findServerByPublicKey(device.Wireguard.PublicKey, managedServersByPublicKey); existingServer != nil {
			return nil, s.serverPublicKeyConflictError(ctx, existingServer)
		}

		var address string
		if len(foreignServer.Interface.Addresses) != 0 {
			address = foreignServer.Interface.Addresses[0]
		}

		createServer, err := s.serverService.CreateServer(ctx, &server.CreateOptions{
			Name:         foreignServer.Interface.Name,
			Description:  foreignServer.Description,
			BackendId:    backendId,
			Enabled:      true,
			Running:      true,
			PrivateKey:   device.Wireguard.PrivateKey,
			ListenPort:   adapt.ToPointerNilZero(device.Wireguard.ListenPort),
			FirewallMark: adapt.ToPointerNilZero(device.Wireguard.FirewallMark),
			Address:      address,
			DNS:          nil,
			MTU:          foreignServer.Interface.Mtu,
		}, userId)
		if err != nil {
			return nil, fmt.Errorf("failed to create server: %w", err)
		}

		foreignPeersByPublicKey := make(map[string]*driver.Peer, len(foreignServer.Peers))
		for _, foreignPeer := range foreignServer.Peers {
			if foreignPeer == nil {
				continue
			}
			publicKey := strings.TrimSpace(foreignPeer.PublicKey)
			if publicKey == "" {
				continue
			}
			if _, exists := foreignPeersByPublicKey[publicKey]; !exists {
				foreignPeersByPublicKey[publicKey] = foreignPeer
			}
		}

		usedPeerNames := make(map[string]struct{}, len(device.Wireguard.Peers))
		for i, p := range device.Wireguard.Peers {
			if p == nil {
				continue
			}

			peerName := strings.TrimSpace(p.Name)
			peerDescription := strings.TrimSpace(p.Description)

			if foreignPeer, exists := foreignPeersByPublicKey[strings.TrimSpace(p.PublicKey)]; exists {
				if peerName == "" {
					peerName = strings.TrimSpace(foreignPeer.Name)
				}
				if peerDescription == "" {
					peerDescription = strings.TrimSpace(foreignPeer.Description)
				}
			}

			_, err := s.peerService.CreatePeer(ctx, createServer.Id, &peer.CreateOptions{
				Name:        importedPeerName(peerName, i, usedPeerNames),
				Description: peerDescription,
				PublicKey:   p.PublicKey,
				Endpoint:    p.Endpoint,
				AllowedIPs: adapt.Array(p.AllowedIPs, func(allowedIp net.IPNet) string {
					return allowedIp.String()
				}),
				PresharedKey:        p.PresharedKey,
				PersistentKeepalive: int(p.PersistentKeepalive.Seconds()),
			}, userId)
			if err != nil {
				return nil, fmt.Errorf("failed to create peer: %w", err)
			}
		}

		return createServer, nil
	})
}

func importedPeerName(preferred string, index int, used map[string]struct{}) string {
	base := normalizeImportedPeerName(preferred)
	if base == "" {
		base = fmt.Sprintf("Peer #%d", index+1)
	}

	if name := reserveUniqueImportedPeerName(base, used); name != "" {
		return name
	}

	fallback := fmt.Sprintf("Peer #%d", index+1)
	if name := reserveUniqueImportedPeerName(fallback, used); name != "" {
		return name
	}

	// This should never happen, but keep deterministic output if we fail to reserve.
	return fallback
}

func normalizeImportedPeerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	name = truncateRunes(name, 30)
	if len([]rune(name)) < 3 {
		return ""
	}

	return name
}

func reserveUniqueImportedPeerName(base string, used map[string]struct{}) string {
	base = normalizeImportedPeerName(base)
	if base == "" {
		return ""
	}

	if used == nil {
		return base
	}

	baseKey := strings.ToLower(base)
	if _, exists := used[baseKey]; !exists {
		used[baseKey] = struct{}{}
		return base
	}

	for suffixIndex := 2; ; suffixIndex++ {
		suffix := fmt.Sprintf(" (%d)", suffixIndex)
		maxBaseLen := 30 - len([]rune(suffix))
		if maxBaseLen < 3 {
			maxBaseLen = 3
		}

		candidate := normalizeImportedPeerName(truncateRunes(base, maxBaseLen) + suffix)
		if candidate == "" {
			continue
		}

		candidateKey := strings.ToLower(candidate)
		if _, exists := used[candidateKey]; exists {
			continue
		}

		used[candidateKey] = struct{}{}
		return candidate
	}
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}

func (s *service) CreatePeer(ctx context.Context, serverId string, options *peer.CreateOptions, userId string) (*peer.Peer, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*peer.Peer, error) {
		createdPeer, err := s.peerService.CreatePeer(ctx, serverId, options, userId)
		if err != nil {
			return nil, err
		}
		return s.configurePeerDevice(ctx, createdPeer, userId)
	})
}

func (s *service) UpdatePeer(ctx context.Context, peerId string, options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask, userId string) (*peer.Peer, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*peer.Peer, error) {
		updatedPeer, err := s.peerService.UpdatePeer(ctx, peerId, options, fieldMask, userId)
		if err != nil {
			return nil, err
		}
		return s.configurePeerDevice(ctx, updatedPeer, userId)
	})
}

func (s *service) DeletePeer(ctx context.Context, peerId string, userId string) (*peer.Peer, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*peer.Peer, error) {
		deletedPeer, err := s.peerService.DeletePeer(ctx, peerId, userId)
		if err != nil {
			return nil, err
		}
		return s.configurePeerDevice(ctx, deletedPeer, userId)
	})
}

func (s *service) PeerStats(ctx context.Context, serverId string, peerPublicKey string) (*driver.PeerStats, error) {
	srv, err := s.findServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	b, err := s.findBackend(ctx, srv.BackendId)
	if err != nil {
		return nil, fmt.Errorf("failed to find backend: %w", err)
	}

	return s.wireguardService.PeerStats(ctx, b, srv.Name, peerPublicKey)
}

func (s *service) ForeignServers(ctx context.Context, backendId string) ([]*driver.ForeignServer, error) {
	b, err := s.findBackend(ctx, backendId)
	if err != nil {
		return nil, fmt.Errorf("failed to find backend: %w", err)
	}

	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find servers: %w", err)
	}
	managedServersByPublicKey := serverByPublicKey(servers, "")

	knownInterfaces := adapt.Array(servers, func(server *server.Server) string {
		return server.Name
	})
	foreignServers, err := s.wireguardService.FindForeignServers(ctx, b, knownInterfaces)
	if err != nil {
		return nil, err
	}
	return filterForeignServersByPublicKey(foreignServers, managedServersByPublicKey), nil
}

func (s *service) ForeignServersAll(ctx context.Context) ([]*driver.ForeignServer, error) {
	backends, err := s.backendService.FindBackends(ctx, &backend.FindOptions{
		Enabled: adapt.ToPointer(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find backends: %w", err)
	}

	allForeignServers := make([]*driver.ForeignServer, 0)
	seenPublicKeys := make(map[string]struct{})
	var errs []error

	for _, b := range backends {
		foreignServers, err := s.ForeignServers(ctx, b.Id)
		if err != nil {
			errs = append(errs, fmt.Errorf("backend %s: %w", b.Name, err))
			continue
		}

		for _, foreignServer := range foreignServers {
			if foreignServer == nil {
				continue
			}

			normalizedPublicKey := normalizePublicKey(foreignServer.PublicKey)
			if normalizedPublicKey != "" {
				if _, seen := seenPublicKeys[normalizedPublicKey]; seen {
					continue
				}
				seenPublicKeys[normalizedPublicKey] = struct{}{}
			}

			allForeignServers = append(allForeignServers, foreignServer)
		}
	}

	if len(errs) > 0 && len(allForeignServers) == 0 {
		return nil, errors.Join(errs...)
	}
	return allForeignServers, nil
}

func (s *service) DeleteBackend(ctx context.Context, backendId string, userId string) (*backend.Backend, error) {
	// First delete the backend from the database
	deletedBackend, err := s.backendService.DeleteBackend(ctx, backendId, userId)
	if err != nil {
		return nil, err
	}

	// Then remove it from the wireguard registry
	if err := s.wireguardService.RemoveBackend(ctx, backendId); err != nil {
		logrus.WithError(err).WithField("backendId", backendId).Warn("failed to remove backend from registry")
	}

	return deletedBackend, nil
}

func (s *service) Close() {
	close(s.stopChan)
	<-s.stoppedChan
}

func (s *service) configurePeerDevice(ctx context.Context, p *peer.Peer, userId string) (*peer.Peer, error) {
	srv, err := s.findServer(ctx, p.ServerId)
	if err != nil {
		return nil, err
	}

	b, err := s.findBackend(ctx, srv.BackendId)
	if err != nil {
		return nil, fmt.Errorf("failed to find backend: %w", err)
	}

	status, err := s.wireguardService.Status(ctx, b, srv.Name)
	if err != nil {
		return nil, err
	}

	if !status {
		return p, nil
	}

	peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
		ServerId: &srv.Id,
	})
	if err != nil {
		return nil, err
	}

	device, err := s.configureDevice(ctx, srv, peers)
	if err != nil {
		return nil, err
	}

	if _, err := s.updateServer(ctx, srv, device, userId); err != nil {
		return nil, err
	}

	return p, nil
}

func (s *service) configureDevice(ctx context.Context, srv *server.Server, peers []*peer.Peer) (*driver.Device, error) {
	b, err := s.findBackend(ctx, srv.BackendId)
	if err != nil {
		return nil, fmt.Errorf("failed to find backend: %w", err)
	}

	if !b.Enabled {
		return nil, fmt.Errorf("backend %s is disabled", b.Name)
	}

	if err := s.ensurePublicKeyUnique(ctx, srv.PublicKey, srv.Id); err != nil {
		return nil, err
	}

	return s.wireguardService.Up(ctx, b, driver.ConfigureOptions{
		InterfaceOptions: driver.InterfaceOptions{
			Name:        srv.Name,
			Description: srv.Description,
			Address:     srv.Address,
			DNS:         srv.DNS,
			Mtu:         srv.MTU,
		},
		WireguardOptions: driver.WireguardOptions{
			PrivateKey:   srv.PrivateKey,
			ListenPort:   srv.ListenPort,
			FirewallMark: srv.FirewallMark,
			Peers: adapt.Array(peers, func(peer *peer.Peer) *driver.PeerOptions {
				return &driver.PeerOptions{
					Name:                peer.Name,
					Description:         peer.Description,
					PublicKey:           peer.PublicKey,
					Endpoint:            peer.Endpoint,
					AllowedIPs:          peer.AllowedIPs,
					PresharedKey:        peer.PresharedKey,
					PersistentKeepalive: peer.PersistentKeepalive,
				}
			}),
		},
	})
}

func (s *service) updateServer(ctx context.Context, srv *server.Server, device *driver.Device, userId string) (*server.Server, error) {
	b, err := s.findBackend(ctx, srv.BackendId)
	if err != nil {
		return nil, fmt.Errorf("failed to find backend: %w", err)
	}

	status, err := s.wireguardService.Status(ctx, b, srv.Name)
	if err != nil {
		return nil, err
	}

	updateOptions := server.UpdateOptions{
		Running:      status,
		PrivateKey:   device.Wireguard.PrivateKey,
		ListenPort:   adapt.ToPointerNilZero(device.Wireguard.ListenPort),
		FirewallMark: adapt.ToPointerNilZero(device.Wireguard.FirewallMark),
		MTU:          device.Interface.Mtu,
	}
	updateFieldMask := server.UpdateFieldMask{
		Running:      srv.Running != status,
		PrivateKey:   !strings.EqualFold(srv.PrivateKey, device.Wireguard.PrivateKey),
		ListenPort:   adapt.Dereference(srv.ListenPort) != device.Wireguard.ListenPort,
		FirewallMark: adapt.Dereference(srv.FirewallMark) != device.Wireguard.FirewallMark,
		MTU:          srv.MTU != device.Interface.Mtu,
	}
	return s.serverService.UpdateServer(ctx, srv.Id, &updateOptions, &updateFieldMask, userId)
}

func (s *service) ensurePublicKeyUnique(ctx context.Context, publicKey string, excludeServerId string) error {
	normalizedPublicKey := normalizePublicKey(publicKey)
	if normalizedPublicKey == "" {
		return nil
	}

	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		return fmt.Errorf("failed to find servers: %w", err)
	}

	if existingServer := findServerByPublicKey(normalizedPublicKey, serverByPublicKey(servers, excludeServerId)); existingServer != nil {
		return s.serverPublicKeyConflictError(ctx, existingServer)
	}
	return nil
}

func (s *service) serverPublicKeyConflictError(ctx context.Context, srv *server.Server) error {
	backendName := strings.TrimSpace(srv.BackendId)
	if backendName == "" {
		backendName = "unknown"
	} else if b, err := s.findBackend(ctx, srv.BackendId); err == nil && b != nil && strings.TrimSpace(b.Name) != "" {
		backendName = b.Name
	}

	return fmt.Errorf(
		"wireguard interface with public key is already managed by server %q on backend %q",
		srv.Name,
		backendName,
	)
}

func filterForeignServersByPublicKey(
	foreignServers []*driver.ForeignServer,
	managedServersByPublicKey map[string]*server.Server,
) []*driver.ForeignServer {
	filteredServers := make([]*driver.ForeignServer, 0, len(foreignServers))
	seenPublicKeys := make(map[string]struct{}, len(foreignServers))

	for _, foreignServer := range foreignServers {
		if foreignServer == nil {
			continue
		}

		normalizedPublicKey := normalizePublicKey(foreignServer.PublicKey)
		if normalizedPublicKey != "" {
			if _, exists := managedServersByPublicKey[normalizedPublicKey]; exists {
				continue
			}
			if _, seen := seenPublicKeys[normalizedPublicKey]; seen {
				continue
			}
			seenPublicKeys[normalizedPublicKey] = struct{}{}
		}

		filteredServers = append(filteredServers, foreignServer)
	}

	return filteredServers
}

func serverByPublicKey(servers []*server.Server, excludeServerId string) map[string]*server.Server {
	serversByPublicKey := make(map[string]*server.Server, len(servers))
	for _, srv := range servers {
		if srv == nil || (excludeServerId != "" && srv.Id == excludeServerId) {
			continue
		}

		normalizedPublicKey := normalizePublicKey(srv.PublicKey)
		if normalizedPublicKey == "" {
			continue
		}

		if _, exists := serversByPublicKey[normalizedPublicKey]; !exists {
			serversByPublicKey[normalizedPublicKey] = srv
		}
	}
	return serversByPublicKey
}

func findServerByPublicKey(publicKey string, serversByPublicKey map[string]*server.Server) *server.Server {
	normalizedPublicKey := normalizePublicKey(publicKey)
	if normalizedPublicKey == "" {
		return nil
	}
	return serversByPublicKey[normalizedPublicKey]
}

func normalizePublicKey(publicKey string) string {
	normalizedPublicKey := strings.TrimSpace(publicKey)
	if normalizedPublicKey == "" {
		return ""
	}

	key, err := wgtypes.ParseKey(normalizedPublicKey)
	if err != nil {
		return normalizedPublicKey
	}
	return key.String()
}

func (s *service) testBackendURL(ctx context.Context, rawURL string) error {
	parsedURL, err := backend.ParseURL(rawURL)
	if err != nil {
		return err
	}

	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	backendInstance, err := driver.Create(testCtx, parsedURL.Type, rawURL)
	if err != nil {
		return fmt.Errorf("failed to connect backend: %w", err)
	}
	defer func() {
		if closeErr := backendInstance.Close(testCtx); closeErr != nil {
			logrus.WithError(closeErr).
				WithField("type", parsedURL.Type).
				Warn("failed to close backend during url validation")
		}
	}()

	if _, err := backendInstance.FindForeignServers(testCtx, nil); err != nil {
		return fmt.Errorf("failed to validate backend url: %w", err)
	}

	return nil
}

func (s *service) findUserById(ctx context.Context, userId string) (*user.User, error) {
	u, err := s.userService.FindUser(ctx, &user.FindOneOptions{
		IdOption: &user.IdOption{
			Id: userId,
		},
	})
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (s *service) findServer(ctx context.Context, serverId string) (*server.Server, error) {
	svc, err := s.serverService.FindServer(ctx, &server.FindOneOptions{
		IdOption: &server.IdOption{
			Id: serverId,
		},
	})
	if err != nil {
		return nil, err
	}
	if svc == nil {
		return nil, server.ErrServerNotFound
	}
	return svc, nil
}

func (s *service) findBackend(ctx context.Context, backendId string) (*backend.Backend, error) {
	b, err := s.backendService.FindBackend(ctx, &backend.FindOneOptions{
		IdOption: &backend.IdOption{
			Id: backendId,
		},
	})
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, backend.ErrBackendNotFound
	}
	return b, nil
}

func (s *service) findPeer(ctx context.Context, peerId string) (*peer.Peer, error) {
	p, err := s.peerService.FindPeer(ctx, &peer.FindOneOptions{
		IdOption: &peer.IdOption{
			Id: peerId,
		},
	})
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, peer.ErrPeerNotFound
	}
	return p, nil
}

func (s *service) cleanup(ctx context.Context) {
	started := time.Now()

	removedOrphanServerUsers := s.cleanupOrphanedUsersFromServers(ctx)
	removedOrphanPeerUsers := s.cleanupOrphanedUsersFromPeers(ctx)
	removedOrphanPeerServers := s.cleanupOrphanedPeersFromServers(ctx)

	if removedOrphanServerUsers > 0 || removedOrphanPeerUsers > 0 || removedOrphanPeerServers > 0 {
		logrus.
			WithField("duration", time.Since(started).String()).
			WithField("removedOrphanServerUsers", removedOrphanServerUsers).
			WithField("removedOrphanPeerUsers", removedOrphanPeerUsers).
			WithField("removedOrphanPeerServers", removedOrphanPeerServers).
			Warn("orphaned data found")
	}
}

func (s *service) cleanupOrphanedUsersFromServers(ctx context.Context) int {
	var count int
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		logrus.WithError(err).Warn("failed to find servers")
	}

	for _, svc := range servers {
		count += s.cleanupOrphanedUserFromServer(ctx, svc)
	}
	return count
}

func (s *service) cleanupOrphanedUsersFromPeers(ctx context.Context) int {
	var count int
	peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{})
	if err != nil {
		logrus.WithError(err).Warn("failed to find peers")
	}

	for _, p := range peers {
		count += s.cleanupOrphanedUserFromPeer(ctx, p)
	}
	return count
}

func (s *service) cleanupOrphanedPeersFromServers(ctx context.Context) int {
	var count int
	peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{})
	if err != nil {
		logrus.WithError(err).Warn("failed to find peers")
	}

	for _, p := range peers {
		count += s.cleanupOrphanedServerFromPeer(ctx, p)
	}
	return count
}

func (s *service) cleanupOrphanedUserFromServer(ctx context.Context, svc *server.Server) int {
	var count int
	if svc.CreateUserId != "" {
		if _, err := s.findUserById(ctx, svc.CreateUserId); errors.Is(err, user.ErrUserNotFound) {
			count++
			updateOptions := server.UpdateOptions{}
			updateFieldMask := server.UpdateFieldMask{CreateUserId: true}
			if _, err := s.serverService.UpdateServer(ctx, svc.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update server")
			}
		}
	}

	if svc.UpdateUserId != "" {
		if _, err := s.findUserById(ctx, svc.UpdateUserId); errors.Is(err, user.ErrUserNotFound) {
			count++
			updateOptions := server.UpdateOptions{}
			updateFieldMask := server.UpdateFieldMask{UpdateUserId: true}
			if _, err := s.serverService.UpdateServer(ctx, svc.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update server")
			}
		}
	}
	return count
}

func (s *service) cleanupOrphanedUserFromPeer(ctx context.Context, p *peer.Peer) int {
	var count int
	if p.CreateUserId != "" {
		if _, err := s.findUserById(ctx, p.CreateUserId); errors.Is(err, user.ErrUserNotFound) {
			count++
			updateOptions := peer.UpdateOptions{}
			updateFieldMask := peer.UpdateFieldMask{CreateUserId: true}
			if _, err := s.peerService.UpdatePeer(ctx, p.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update peer")
			}
		}
	}

	if p.UpdateUserId != "" {
		if _, err := s.findUserById(ctx, p.UpdateUserId); errors.Is(err, user.ErrUserNotFound) {
			count++
			updateOptions := peer.UpdateOptions{}
			updateFieldMask := peer.UpdateFieldMask{UpdateUserId: true}
			if _, err := s.peerService.UpdatePeer(ctx, p.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update peer")
			}
		}
	}
	return count
}

func (s *service) cleanupOrphanedServerFromPeer(ctx context.Context, p *peer.Peer) int {
	var count int
	if _, err := s.findServer(ctx, p.ServerId); errors.Is(err, server.ErrServerNotFound) {
		count++
		if _, err := s.peerService.DeletePeer(ctx, p.Id, ""); err != nil {
			logrus.WithError(err).Warn("failed to delete peer")
		}
	}
	return count
}

func (s *service) updateServersStats(ctx context.Context) {
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		logrus.
			WithError(err).
			Error("failed to find servers")
		return
	}

	for _, srv := range servers {
		if err := s.updateServerStats(ctx, srv); err != nil {
			logrus.
				WithError(err).
				WithField("name", srv.Name).
				Warn("failed to get interface stats")
			continue
		}
	}
}

func (s *service) updateServerStats(ctx context.Context, srv *server.Server) error {
	if !srv.Enabled || !srv.Running {
		return nil
	}

	b, err := s.findBackend(ctx, srv.BackendId)
	if err != nil {
		return fmt.Errorf("failed to find backend: %w", err)
	}

	stats, err := s.wireguardService.Stats(ctx, b, srv.Name)
	if err != nil {
		return fmt.Errorf("failed to get device stats: %w", err)
	}
	if stats == nil {
		stats = &driver.InterfaceStats{}
	}

	newStats := server.Stats{
		RxBytes: stats.RxBytes,
		TxBytes: stats.TxBytes,
	}

	if newStats != srv.Stats {
		updateOptions := &server.UpdateOptions{Stats: newStats}
		updateFieldMask := &server.UpdateFieldMask{Stats: true}
		if _, err = s.serverService.UpdateServer(ctx, srv.Id, updateOptions, updateFieldMask, ""); err != nil {
			return fmt.Errorf("failed to update server stats: %w", err)
		}
	}
	return nil
}
