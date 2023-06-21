package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

func (r *mutationResolver) DeleteServer(ctx context.Context, input model.DeleteServerInput) (*model.DeleteServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	stoppedServer, err := r.wgService.StopServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	deletedServer, err := r.serverService.DeleteServer(ctx, stoppedServer.Id, userId)
	if err != nil {
		return nil, err
	}

	s := model.ToServer(deletedServer)
	serverChanged := &model.ServerChangedEvent{
		Node:   s,
		Action: model.ServerActionDeleted,
	}
	if err := r.serverSubscriptionService.Notify(serverChanged); err != nil {
		return nil, err
	}

	return &model.DeleteServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           s,
	}, nil
}
