package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/user"
)

func (r *queryResolver) Users(ctx context.Context, query *string) ([]*model.User, error) {
	users, err := r.userService.FindUsers(ctx, &user.FindOptions{
		Query: adapt.Dereference(query),
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(users, model.ToUser), nil
}
