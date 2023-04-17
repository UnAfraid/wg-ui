package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/server"
)

func (r *queryResolver) Servers(ctx context.Context, query *string, enabled *bool) ([]*model.Server, error) {
	servers, err := r.serverService.FindServers(ctx, &server.FindOptions{
		Ids:     nil,
		Query:   adapt.Dereference(query),
		Enabled: enabled,
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(servers, model.ToServer), nil
}
