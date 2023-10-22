package manage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/UnAfraid/wg-ui/pkg/dbx"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
	"github.com/UnAfraid/wg-ui/pkg/wireguard"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

type Service interface {
	Authenticate(ctx context.Context, username string, password string) (*user.User, error)
	CreateUser(ctx context.Context, options *user.CreateOptions) (*user.User, error)
	UpdateUser(ctx context.Context, userId string, options *user.UpdateOptions, fieldMask *user.UpdateFieldMask) (*user.User, error)
	DeleteUser(ctx context.Context, userId string) (*user.User, error)
	CreateServer(ctx context.Context, options *server.CreateOptions, userId string) (*server.Server, error)
	UpdateServer(ctx context.Context, serverId string, options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, userId string) (*server.Server, error)
	DeleteServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	StartServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	StopServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	ImportForeignServer(ctx context.Context, name string, userId string) (*server.Server, error)
	CreatePeer(ctx context.Context, serverId string, options *peer.CreateOptions, userId string) (*peer.Peer, error)
	UpdatePeer(ctx context.Context, peerId string, options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask, userId string) (*peer.Peer, error)
	DeletePeer(ctx context.Context, peerId string, userId string) (*peer.Peer, error)
	PeerStats(ctx context.Context, name string, peerPublicKey string) (*backend.PeerStats, error)
	ForeignServers(ctx context.Context) ([]*backend.ForeignServer, error)
}

type service struct {
	transactionScoper dbx.TransactionScoper
	userService       user.Service
	serverService     server.Service
	peerService       peer.Service
	wireguardService  wireguard.Service
}

func NewService(
	transactionScoper dbx.TransactionScoper,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	wireguardService wireguard.Service,
) Service {
	s := &service{
		transactionScoper: transactionScoper,
		userService:       userService,
		serverService:     serverService,
		peerService:       peerService,
		wireguardService:  wireguardService,
	}

	s.cleanup(context.Background())
	s.init()

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

	for _, srv := range servers {
		peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{
			ServerId: adapt.ToPointer(srv.Id),
		})
		if err != nil {
			logrus.WithError(err).WithField("name", srv.Name).Error("failed to find peers for server")
			return
		}

		if _, err = s.configureDevice(ctx, srv, peers); err != nil {
			logrus.WithError(err).WithField("name", srv.Name).Error("failed to configure wireguard device")
			return
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

		if !updatedServer.Enabled {
			status, err := s.wireguardService.Status(ctx, updatedServer.Name)
			if err != nil {
				return nil, err
			}

			if status {
				if err := s.wireguardService.Down(ctx, updatedServer.Name); err != nil {
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

		return updatedServer, nil
	})
}

func (s *service) DeleteServer(ctx context.Context, serverId string, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		svc, err := s.findServer(ctx, serverId)
		if err != nil {
			return nil, err
		}

		if err = s.wireguardService.Down(ctx, svc.Name); err != nil {
			logrus.
				WithError(err).
				WithField("serverId", svc.Id).
				WithField("serverName", svc.Name).
				Warn("failed to stop server")
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

		if err := s.wireguardService.Down(ctx, srv.Name); err != nil {
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

func (s *service) ImportForeignServer(ctx context.Context, name string, userId string) (*server.Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*server.Server, error) {
		servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to find servers: %w", err)
		}

		knownInterfaces := adapt.Array(servers, func(server *server.Server) string {
			return server.Name
		})

		foreignInterfaces, err := s.wireguardService.FindForeignServers(ctx, knownInterfaces)
		if err != nil {
			return nil, fmt.Errorf("failed to find foreign interfaces: %w", err)
		}

		var foreignInterface *backend.ForeignInterface
		for _, fn := range foreignInterfaces {
			if strings.EqualFold(fn.Name, name) {
				foreignInterface = fn.Interface
				break
			}
		}

		if foreignInterface == nil {
			return nil, fmt.Errorf("foreign interface: %s not found", name)
		}

		device, err := s.wireguardService.Device(ctx, foreignInterface.Name)
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
			Running:      true,
			PrivateKey:   device.Wireguard.PrivateKey,
			ListenPort:   adapt.ToPointerNilZero(device.Wireguard.ListenPort),
			FirewallMark: adapt.ToPointerNilZero(device.Wireguard.FirewallMark),
			Address:      address,
			DNS:          nil,
			MTU:          foreignInterface.Mtu,
		}, userId)
		if err != nil {
			return nil, fmt.Errorf("failed to create server: %w", err)
		}

		for i, p := range device.Wireguard.Peers {
			_, err := s.peerService.CreatePeer(ctx, createServer.Id, &peer.CreateOptions{
				Name:        fmt.Sprintf("Peer #%d", i+1),
				Description: "",
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

func (s *service) PeerStats(ctx context.Context, name string, peerPublicKey string) (*backend.PeerStats, error) {
	return s.wireguardService.PeerStats(ctx, name, peerPublicKey)
}

func (s *service) ForeignServers(ctx context.Context) ([]*backend.ForeignServer, error) {
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to find servers: %w", err)
	}

	knownInterfaces := adapt.Array(servers, func(server *server.Server) string {
		return server.Name
	})
	return s.wireguardService.FindForeignServers(ctx, knownInterfaces)
}

func (s *service) configurePeerDevice(ctx context.Context, p *peer.Peer, userId string) (*peer.Peer, error) {
	srv, err := s.findServer(ctx, p.ServerId)
	if err != nil {
		return nil, err
	}

	status, err := s.wireguardService.Status(ctx, srv.Name)
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

func (s *service) configureDevice(ctx context.Context, srv *server.Server, peers []*peer.Peer) (*backend.Device, error) {
	return s.wireguardService.Up(ctx, backend.ConfigureOptions{
		InterfaceOptions: backend.InterfaceOptions{
			Name:    srv.Name,
			Address: srv.Address,
			Mtu:     srv.MTU,
		},
		WireguardOptions: backend.WireguardOptions{
			PrivateKey:   srv.PrivateKey,
			ListenPort:   srv.ListenPort,
			FirewallMark: srv.FirewallMark,
			Peers: adapt.Array(peers, func(peer *peer.Peer) *backend.PeerOptions {
				return &backend.PeerOptions{
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

func (s *service) updateServer(ctx context.Context, srv *server.Server, device *backend.Device, userId string) (*server.Server, error) {
	status, err := s.wireguardService.Status(ctx, srv.Name)
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
