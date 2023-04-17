package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *queryResolver) Viewer(ctx context.Context) (*model.User, error) {
	return model.ContextToUser(ctx)
}
