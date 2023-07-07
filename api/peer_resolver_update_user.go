package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/handler"
	"github.com/UnAfraid/wg-ui/api/model"
)

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
