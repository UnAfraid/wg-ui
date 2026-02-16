package server

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
)

type serverResolver struct {
	peerService peer.Service
}

func NewServerResolver(
	peerService peer.Service,
) resolver.ServerResolver {
	return &serverResolver{
		peerService: peerService,
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

func (r *serverResolver) Backend(ctx context.Context, srv *model.Server) (*model.Backend, error) {
	if srv.Backend == nil {
		return nil, nil
	}

	backendId, err := srv.Backend.ID.String(model.IdKindBackend)
	if err != nil {
		return nil, err
	}

	backendLoader, err := handler.BackendLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return backendLoader.Load(ctx, backendId)()
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
