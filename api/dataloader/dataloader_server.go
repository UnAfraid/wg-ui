package dataloader

import (
	"context"
	"errors"

	"github.com/UnAfraid/dataloaden/v2/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/server"
)

//go:generate go run github.com/UnAfraid/dataloaden/v2 -name ServerLoader -keyType string -valueType *github.com/UnAfraid/wg-ui/api/model.Server
var serverLoaderCtxKey = &contextKey{"serverLoader"}

func ServerLoaderFromContext(ctx context.Context) (dataloader.DataLoader[string, *model.Server], error) {
	dataLoader, ok := ctx.Value(serverLoaderCtxKey).(dataloader.DataLoader[string, *model.Server])
	if !ok {
		return nil, errors.New("server loader not found")
	}
	return dataLoader, nil
}

func serverFetcher(ctx context.Context, serverService server.Service) func([]string) ([]*model.Server, []error) {
	return func(ids []string) ([]*model.Server, []error) {
		servers, err := serverService.FindServers(ctx, &server.FindOptions{
			Ids: ids,
		})
		if err != nil {
			return nil, repeatError(err, len(ids))
		}
		return adapt.Array(servers, model.ToServer), nil
	}
}
