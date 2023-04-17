package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *serverResolver) CreateUser(ctx context.Context, srv *model.Server) (*model.User, error) {
	if srv.CreateUser == nil {
		return nil, nil
	}

	userId, err := srv.CreateUser.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	userLoader, err := dataloader.UserLoaderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return userLoader.Load(userId)
}
