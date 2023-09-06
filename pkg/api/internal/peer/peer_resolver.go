package peer

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/wg"
)

type peerResolver struct {
	peerService peer.Service
	wgService   wg.Service
}

func NewPeerResolver(
	peerService peer.Service,
	wgService wg.Service,
) resolver.PeerResolver {
	return &peerResolver{
		wgService:   wgService,
		peerService: peerService,
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

	stats, err := r.wgService.PeerStats(server.Name, p.PublicKey)
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
