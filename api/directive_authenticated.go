package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type authenticated struct {
	userService user.Service
}

func (a *authenticated) directive(ctx context.Context, _ interface{}, next graphql.Resolver) (res interface{}, err error) {
	token, claims, err := jwtauth.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("access denied: %v", err)
	}

	if token == nil {
		return nil, fmt.Errorf("access denied: %v", ErrAuthenticationRequired)
	}

	ctx, err = a.processToken(ctx, token, claims)
	if err != nil {
		return nil, fmt.Errorf("access denied: %v", err)
	}

	return next(ctx)
}

func (a *authenticated) processToken(ctx context.Context, token jwt.Token, claims map[string]interface{}) (context.Context, error) {
	if err := jwt.Validate(token); err != nil {
		return nil, err
	}

	userIdValue, ok := claims["userId"]
	if !ok {
		return nil, errors.New("failed to parse jwt claims key userId is missing")
	}

	userId, ok := userIdValue.(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse jwt claims key userId should be a string")
	}

	u, err := a.processUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, ErrUserNotFound
	}

	return model.UserToContext(ctx, u)
}

func (a *authenticated) processUserId(ctx context.Context, userID string) (*model.User, error) {
	var id model.ID
	if err := id.UnmarshalGQL(userID); err != nil {
		return nil, err
	}

	userId, err := id.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}
	return a.processUserAuth(ctx, userId)
}

func (a *authenticated) processUserAuth(ctx context.Context, userId string) (*model.User, error) {
	u, err := a.userService.FindUser(ctx, &user.FindOneOptions{
		IdOption: &user.IdOption{
			Id: userId,
		},
	})
	if err != nil {
		return nil, err
	}
	return model.ToUser(u), nil
}
