package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *peerResolver) DeleteUser(ctx context.Context, p *model.Peer) (*model.User, error) {
	if p.DeleteUser == nil {
		return nil, nil
	}

	userId, err := p.DeleteUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := dataloader.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(userId)
}
