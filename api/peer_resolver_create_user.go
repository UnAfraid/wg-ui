package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/handler"
	"github.com/UnAfraid/wg-ui/api/model"
)

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
