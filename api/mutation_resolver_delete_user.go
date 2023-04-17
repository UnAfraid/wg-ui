package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) DeleteUser(ctx context.Context, input model.DeleteUserInput) (*model.DeleteUserPayload, error) {
	userId, err := input.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	deletedUser, err := r.userService.DeleteUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	u := model.ToUser(deletedUser)
	userChanged := &model.UserChangedEvent{
		Node:   u,
		Action: model.UserActionDeleted,
	}
	if err := r.userSubscriptionService.Notify(userChanged); err != nil {
		return nil, err
	}

	return &model.DeleteUserPayload{
		ClientMutationID: input.ClientMutationID,
		User:             u,
	}, nil
}
