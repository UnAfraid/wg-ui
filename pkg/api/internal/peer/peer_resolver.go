package peer

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/manage"
)

type peerResolver struct {
	manageService manage.Service
}

func NewPeerResolver(
	manageService manage.Service,
) resolver.PeerResolver {
	return &peerResolver{
		manageService: manageService,
	}
}

func (r *peerResolver) Server(ctx context.Context, p *model.Peer) (*model.Server, error) {
	if p.Server == nil {
		return nil, nil
	}

	serverId, err := p.Server.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	serverLoader, err := handler.ServerLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return serverLoader.Load(ctx, serverId)()
}

func (r *peerResolver) Backend(ctx context.Context, p *model.Peer) (*model.Backend, error) {
	// Backend is resolved through the server
	if p.Server == nil {
		return nil, nil
	}

	serverId, err := p.Server.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	serverLoader, err := handler.ServerLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	srv, err := serverLoader.Load(ctx, serverId)()
	if err != nil {
		return nil, err
	}

	if srv == nil || srv.Backend == nil {
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

func (r *peerResolver) Stats(ctx context.Context, p *model.Peer) (*model.PeerStats, error) {
	if p.Server == nil {
		return nil, nil
	}

	serverId, err := p.Server.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	serverLoader, err := handler.ServerLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	server, err := serverLoader.Load(ctx, serverId)()
	if err != nil {
		return nil, err
	}

	if !server.Running {
		return nil, nil
	}

	stats, err := r.manageService.PeerStats(ctx, serverId, p.PublicKey)
	if err != nil {
		return nil, err
	}

	return model.ToPeerStats(stats), nil
}

func (r *peerResolver) CreateUser(ctx context.Context, p *model.Peer) (*model.User, error) {
	if p.CreateUser == nil {
		return nil, nil
	}

	userId, err := p.CreateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func (r *peerResolver) UpdateUser(ctx context.Context, p *model.Peer) (*model.User, error) {
	if p.UpdateUser == nil {
		return nil, nil
	}

	userId, err := p.UpdateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}

func (r *peerResolver) DeleteUser(ctx context.Context, p *model.Peer) (*model.User, error) {
	if p.DeleteUser == nil {
		return nil, nil
	}

	userId, err := p.DeleteUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := handler.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(ctx, userId)()
}
