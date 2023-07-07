package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/handler"
	"github.com/UnAfraid/wg-ui/api/model"
)

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
