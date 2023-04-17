package api

import (
	"context"
	"fmt"

	"github.com/UnAfraid/wg-ui/api/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *queryResolver) Node(ctx context.Context, id model.ID) (model.Node, error) {
	switch id.Kind {
	case model.IdKindUser:
		userLoader, err := dataloader.UserLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return userLoader.Load(id.Value)
	case model.IdKindServer:
		serverLoader, err := dataloader.ServerLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return serverLoader.Load(id.Value)
	case model.IdKindPeer:
		peerLoader, err := dataloader.PeerLoaderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return peerLoader.Load(id.Value)
	default:
		return nil, fmt.Errorf("node type %s is %w", id.Kind, ErrNotImplemented)
	}
}
