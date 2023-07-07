package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/handler"
	"github.com/UnAfraid/wg-ui/api/model"
)

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
