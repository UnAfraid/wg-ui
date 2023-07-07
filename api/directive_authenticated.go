package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/UnAfraid/wg-ui/api/model"
)

func authenticated(ctx context.Context, _ interface{}, next graphql.Resolver) (res interface{}, err error) {
	_, err = model.ContextToUser(ctx)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, fmt.Errorf("access denied: authentication required")
		}
		return nil, fmt.Errorf("access denied: %v", err)
	}
	return next(ctx)
}
