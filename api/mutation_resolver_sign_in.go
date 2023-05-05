package api

import (
	"context"
	"time"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) SignIn(ctx context.Context, input model.SignInInput) (*model.SignInPayload, error) {
	u, err := r.userService.Authenticate(ctx, input.Email, input.Password)
	if err != nil {
		return nil, err
	}
	user := model.ToUser(u)

	expiresAt := time.Now().Add(r.jwtDuration)
	_, tokenString, err := r.jwtAuth.Encode(map[string]interface{}{
		"userId": user.ID.Base64(),
		"iat":    time.Now().Unix(),
		"exp":    expiresAt.Unix(),
	})
	if err != nil {
		return nil, err
	}

	return &model.SignInPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Token:            tokenString,
		ExpiresAt:        expiresAt,
		ExpiresIn:        int(r.jwtDuration.Seconds()),
	}, nil
}
