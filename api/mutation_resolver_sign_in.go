package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) SignIn(ctx context.Context, input model.SignInInput) (*model.SignInPayload, error) {
	u, err := r.userService.Authenticate(ctx, input.Email, input.Password)
	if err != nil {
		return nil, err
	}
	user := model.ToUser(u)

	tokenString, expiresIn, expiresAt, err := r.authService.Sign(user.ID.Base64())
	if err != nil {
		return nil, err
	}

	return &model.SignInPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Token:            tokenString,
		ExpiresAt:        expiresAt,
		ExpiresIn:        int(expiresIn.Seconds()),
	}, nil
}
