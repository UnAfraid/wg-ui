package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) CreateServer(ctx context.Context, input model.CreateServerInput) (*model.CreateServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	createOptions, err := model.CreateServerInputToCreateServerOptions(input)
	if err != nil {
		return nil, err
	}

	createdServer, err := r.serverService.CreateServer(ctx, createOptions, userId)
	if err != nil {
		return nil, err
	}

	s := model.ToServer(createdServer)
	serverChanged := &model.ServerChangedEvent{
		Node:   s,
		Action: model.ServerActionCreated,
	}
	if err := r.serverSubscriptionService.Notify(serverChanged); err != nil {
		return nil, err
	}

	return &model.CreateServerPayload{
		ClientMutationID: input.ClientMutationID,
		Server:           s,
	}, nil
}
