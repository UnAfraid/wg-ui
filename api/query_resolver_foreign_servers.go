package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
)

func (r *queryResolver) ForeignServers(ctx context.Context) ([]*model.ForeignServer, error) {
	foreignServers, err := r.wgService.ForeignServers(ctx)
	if err != nil {
		return nil, err
	}
	return adapt.Array(foreignServers, model.ToForeignServer), nil
}
