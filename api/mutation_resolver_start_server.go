package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/sirupsen/logrus"
)

func (r *mutationResolver) StartServer(ctx context.Context, input model.StartServerInput) (*model.StartServerPayload, error) {
	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	srv, err := r.wgService.StartServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	if err := srv.RunHooks(server.HookActionStart); err != nil {
		logrus.
			WithError(err).
			WithField("server", srv.Name).
			Error("failed to run hooks on server start")
	}

	s := model.ToServer(srv)
	serverChanged := &model.ServerChangedEvent{
		Node:   s,
		Action: model.ServerActionStarted,
	}
	if err := r.serverSubscriptionService.Notify(serverChanged); err != nil {
		return nil, err
	}

	return &model.StartServerPayload{
		ClientMutationID: input.ClientMutationID,
		Server:           s,
	}, nil
}
