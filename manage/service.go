package manage

import (
	"context"
	"errors"
	"time"

	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
	"github.com/sirupsen/logrus"
)

type Service interface {
	Authenticate(ctx context.Context, username string, password string) (*user.User, error)
	CreateUser(ctx context.Context, options *user.CreateOptions) (*user.User, error)
	UpdateUser(ctx context.Context, userId string, options *user.UpdateOptions, fieldMask *user.UpdateFieldMask) (*user.User, error)
	DeleteUser(ctx context.Context, userId string) (*user.User, error)
	CreateServer(ctx context.Context, options *server.CreateOptions, userId string) (*server.Server, error)
	UpdateServer(ctx context.Context, serverId string, options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, userId string) (*server.Server, error)
	DeleteServer(ctx context.Context, serverId string, userId string) (*server.Server, error)
	StartServer(ctx context.Context, serverId string) (*server.Server, error)
	StopServer(ctx context.Context, serverId string) (*server.Server, error)
	ImportForeignServer(ctx context.Context, name string, userId string) (*server.Server, error)
	CreatePeer(ctx context.Context, serverId string, options *peer.CreateOptions, userId string) (*peer.Peer, error)
	UpdatePeer(ctx context.Context, peerId string, options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask, userId string) (*peer.Peer, error)
	DeletePeer(ctx context.Context, peerId string, userId string) (*peer.Peer, error)
}

type service struct {
	userService   user.Service
	serverService server.Service
	peerService   peer.Service
	wgService     wg.Service
}

func NewService(
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	wgService wg.Service,
) Service {
	s := &service{
		userService:   userService,
		serverService: serverService,
		peerService:   peerService,
		wgService:     wgService,
	}

	s.cleanup(context.Background())

	return s
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
}

func (s *service) CreateServer(ctx context.Context, options *server.CreateOptions, userId string) (*server.Server, error) {
	createdServer, err := s.serverService.CreateServer(ctx, options, userId)
	if err != nil {
		return nil, err
	}

	if createdServer.Enabled {
		return s.wgService.StartServer(ctx, createdServer.Id)
	}

	return createdServer, nil
}

func (s *service) UpdateServer(ctx context.Context, serverId string, options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, userId string) (*server.Server, error) {
	updatedServer, err := s.serverService.UpdateServer(ctx, serverId, options, fieldMask, userId)
	if err != nil {
		return nil, err
	}

	if !updatedServer.Enabled {
		return s.wgService.StopServer(ctx, updatedServer.Id)
	}

	return updatedServer, nil
}

func (s *service) DeleteServer(ctx context.Context, serverId string, userId string) (*server.Server, error) {
	svc, err := s.findServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	if _, err = s.wgService.StopServer(ctx, svc.Id); err != nil {
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
		if _, err := s.peerService.DeletePeer(ctx, p.Id, userId); err != nil {
			logrus.
				WithError(err).
				WithField("serverId", svc.Id).
				WithField("serverName", svc.Name).
				WithField("peerId", p.Id).
				WithField("peerName", p.Name).
				Warn("failed to delete peer")
		}
	}

	return deletedServer, nil
}

func (s *service) StartServer(ctx context.Context, serverId string) (*server.Server, error) {
	return s.wgService.StartServer(ctx, serverId)
}

func (s *service) StopServer(ctx context.Context, serverId string) (*server.Server, error) {
	return s.wgService.StopServer(ctx, serverId)
}

func (s *service) ImportForeignServer(ctx context.Context, name string, userId string) (*server.Server, error) {
	return s.wgService.ImportForeignServer(ctx, name, userId)
}

func (s *service) CreatePeer(ctx context.Context, serverId string, options *peer.CreateOptions, userId string) (*peer.Peer, error) {
	createdPeer, err := s.peerService.CreatePeer(ctx, serverId, options, userId)
	if err != nil {
		return nil, err
	}

	if err := s.wgService.AddPeer(ctx, createdPeer.Id); err != nil {
		logrus.
			WithError(err).
			WithField("peerId", createdPeer.Id).
			WithField("peerName", createdPeer.Name).
			Warn("failed to add peer")
	}

	return createdPeer, nil
}

func (s *service) UpdatePeer(ctx context.Context, peerId string, options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask, userId string) (*peer.Peer, error) {
	updatedPeer, err := s.peerService.UpdatePeer(ctx, peerId, options, fieldMask, userId)
	if err != nil {
		return nil, err
	}

	if err := s.wgService.UpdatePeer(ctx, peerId); err != nil {
		logrus.
			WithError(err).
			WithField("peerId", updatedPeer.Id).
			WithField("peerName", updatedPeer.Name).
			Warn("failed to update peer")
	}

	return updatedPeer, nil
}

func (s *service) DeletePeer(ctx context.Context, peerId string, userId string) (*peer.Peer, error) {
	p, err := s.findPeer(ctx, peerId)
	if err != nil {
		return nil, err
	}

	if err := s.wgService.RemovePeer(ctx, peerId); err != nil {
		logrus.
			WithError(err).
			WithField("peerId", p.Id).
			WithField("peerName", p.Name).
			Warn("failed to remove peer")
	}

	deletedPeer, err := s.peerService.DeletePeer(ctx, peerId, userId)
	if err != nil {
		return nil, err
	}

	return deletedPeer, nil
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

	removedOrphanServerUsers := s.cleanupServers(ctx)
	removedOrphanPeerUsers := s.cleanupPeers(ctx)

	if removedOrphanServerUsers > 0 || removedOrphanPeerUsers > 0 {
		logrus.
			WithField("removedOrphanServerUsers", removedOrphanServerUsers).
			WithField("removedOrphanPeerUsers", removedOrphanPeerUsers).
			WithField("duration", time.Since(started).String()).
			Warn("orphaned users found")
	}
}

func (s *service) cleanupServers(ctx context.Context) int {
	var removedOrphanServerUsers int
	servers, err := s.serverService.FindServers(ctx, &server.FindOptions{})
	if err != nil {
		logrus.WithError(err).Warn("failed to find servers")
	}

	for _, svc := range servers {
		removedOrphanServerUsers += s.cleanupOrphanedUserFromServer(ctx, svc)
	}
	return removedOrphanServerUsers
}

func (s *service) cleanupPeers(ctx context.Context) int {
	var removedOrphanPeerUsers int
	peers, err := s.peerService.FindPeers(ctx, &peer.FindOptions{})
	if err != nil {
		logrus.WithError(err).Warn("failed to find peers")
	}

	for _, p := range peers {
		removedOrphanPeerUsers += s.cleanupOrphanedUserFromPeer(ctx, p)
	}
	return removedOrphanPeerUsers
}

func (s *service) cleanupOrphanedUserFromServer(ctx context.Context, svc *server.Server) int {
	var removedOrphanedUsers int
	if svc.CreateUserId != "" {
		if _, err := s.findUserById(ctx, svc.CreateUserId); errors.Is(err, user.ErrUserNotFound) {
			removedOrphanedUsers++
			updateOptions := server.UpdateOptions{}
			updateFieldMask := server.UpdateFieldMask{CreateUserId: true}
			if _, err := s.serverService.UpdateServer(ctx, svc.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update server")
			}
		}
	}

	if svc.UpdateUserId != "" {
		if _, err := s.findUserById(ctx, svc.UpdateUserId); errors.Is(err, user.ErrUserNotFound) {
			removedOrphanedUsers++
			updateOptions := server.UpdateOptions{}
			updateFieldMask := server.UpdateFieldMask{UpdateUserId: true}
			if _, err := s.serverService.UpdateServer(ctx, svc.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update server")
			}
		}
	}
	return removedOrphanedUsers
}

func (s *service) cleanupOrphanedUserFromPeer(ctx context.Context, p *peer.Peer) int {
	var removedOrphanedUsers int
	if p.CreateUserId != "" {
		if _, err := s.findUserById(ctx, p.CreateUserId); errors.Is(err, user.ErrUserNotFound) {
			removedOrphanedUsers++
			updateOptions := peer.UpdateOptions{}
			updateFieldMask := peer.UpdateFieldMask{CreateUserId: true}
			if _, err := s.peerService.UpdatePeer(ctx, p.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update peer")
			}
		}
	}

	if p.UpdateUserId != "" {
		if _, err := s.findUserById(ctx, p.UpdateUserId); errors.Is(err, user.ErrUserNotFound) {
			removedOrphanedUsers++
			updateOptions := peer.UpdateOptions{}
			updateFieldMask := peer.UpdateFieldMask{UpdateUserId: true}
			if _, err := s.peerService.UpdatePeer(ctx, p.Id, &updateOptions, &updateFieldMask, ""); err != nil {
				logrus.WithError(err).Warn("failed to update peer")
			}
		}
	}
	return removedOrphanedUsers
}
