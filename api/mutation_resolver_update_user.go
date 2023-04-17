package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*model.UpdateUserPayload, error) {
	updateOptions, updateFieldMask, err := model.UpdateUserInputToUserUpdateUserOptions(ctx, input)
	if err != nil {
		return nil, err
	}

	userId, err := input.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	updatedUser, err := r.userService.UpdateUser(ctx, userId, updateOptions, updateFieldMask)
	if err != nil {
		return nil, err
	}

	u := model.ToUser(updatedUser)
	userChanged := &model.UserChangedEvent{
		Node:   u,
		Action: model.UserActionUpdated,
	}
	if err := r.userSubscriptionService.Notify(userChanged); err != nil {
		return nil, err
	}

	return &model.UpdateUserPayload{
		ClientMutationID: input.ClientMutationID,
		User:             u,
	}, nil
}
