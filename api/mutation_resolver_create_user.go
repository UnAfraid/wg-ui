package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.CreateUserPayload, error) {
	createdUser, err := r.userService.CreateUser(ctx, model.CreateUserInputToUserCreateUserOptions(input))
	if err != nil {
		return nil, err
	}

	u := model.ToUser(createdUser)
	userChanged := &model.UserChangedEvent{
		Node:   u,
		Action: model.UserActionCreated,
	}
	if err := r.userSubscriptionService.Notify(userChanged); err != nil {
		return nil, err
	}

	return &model.CreateUserPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		User:             u,
	}, nil
}
