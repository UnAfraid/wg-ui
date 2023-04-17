package dataloader

import (
	"context"
	"errors"

	"github.com/UnAfraid/dataloaden/v2/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/user"
)

//go:generate go run github.com/UnAfraid/dataloaden/v2 -name UserLoader -keyType string -valueType *github.com/UnAfraid/wg-ui/api/model.User
var userLoaderCtxKey = &contextKey{"userLoader"}

func UserLoaderFromContext(ctx context.Context) (dataloader.DataLoader[string, *model.User], error) {
	dataLoader, ok := ctx.Value(userLoaderCtxKey).(dataloader.DataLoader[string, *model.User])
	if !ok {
		return nil, errors.New("user loader not found")
	}
	return dataLoader, nil
}

func userFetcher(ctx context.Context, userService user.Service) func([]string) ([]*model.User, []error) {
	return func(ids []string) ([]*model.User, []error) {
		users, err := userService.FindUsers(ctx, &user.FindOptions{
			Ids: ids,
		})
		if err != nil {
			return nil, repeatError(err, len(ids))
		}
		return adapt.Array(users, model.ToUser), nil
	}
}
