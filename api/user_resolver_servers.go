package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/server"
)

func (r *userResolver) Servers(ctx context.Context, u *model.User) ([]*model.Server, error) {
	userId, err := u.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	servers, err := r.serverService.FindServers(ctx, &server.FindOptions{
		CreateUserId: &userId,
	})
	if err != nil {
		return nil, err
	}

	return adapt.Array(servers, model.ToServer), nil
}
