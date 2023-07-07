package api

import (
	"context"
	"fmt"

	"github.com/UnAfraid/wg-ui/api/handler"
	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *queryResolver) Node(ctx context.Context, id model.ID) (model.Node, error) {
	switch id.Kind {
	case model.IdKindUser:
		userLoader, err := handler.UserLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return userLoader.Load(ctx, id.Value)()
	case model.IdKindServer:
		serverLoader, err := handler.ServerLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return serverLoader.Load(ctx, id.Value)()
	case model.IdKindPeer:
		peerLoader, err := handler.PeerLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return peerLoader.Load(ctx, id.Value)()
	default:
		return nil, fmt.Errorf("node type %s is %w", id.Kind, ErrNotImplemented)
	}
}
