package server

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/internal/handler"
	"github.com/UnAfraid/wg-ui/api/internal/model"
	"github.com/UnAfraid/wg-ui/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/wg"
)

type serverResolver struct {
	serverService server.Service
	peerService   peer.Service
	wgService     wg.Service
}

func NewServerResolver(
	serverService server.Service,
	peerService peer.Service,
	wgService wg.Service,
) resolver.ServerResolver {
	return &serverResolver{
		serverService: serverService,
		peerService:   peerService,
		wgService:     wgService,
	}
}

func (r *serverResolver) Peers(ctx context.Context, svc *model.Server) ([]*model.Peer, error) {
	serverId, err := svc.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		ServerId: &serverId,
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(peers, model.ToPeer), nil
}

func (r *serverResolver) CreateUser(ctx context.Context, srv *model.Server) (*model.User, error) {
	if srv.CreateUser == nil {
		return nil, nil
	}

	userId, err := srv.CreateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func (r *serverResolver) UpdateUser(ctx context.Context, srv *model.Server) (*model.User, error) {
	if srv.UpdateUser == nil {
		return nil, nil
	}

	userId, err := srv.UpdateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func (r *serverResolver) DeleteUser(ctx context.Context, srv *model.Server) (*model.User, error) {
	if srv.DeleteUser == nil {
		return nil, nil
	}

	userId, err := srv.DeleteUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}
