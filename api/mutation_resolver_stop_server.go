package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/sirupsen/logrus"
)

func (r *mutationResolver) StopServer(ctx context.Context, input model.StopServerInput) (*model.StopServerPayload, error) {
	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	srv, err := r.wgService.StopServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	if err := srv.RunHooks(server.HookActionStop); err != nil {
		logrus.
			WithError(err).
			WithField("server", srv.Name).
			Error("failed to run hooks on server stop")
	}

	s := model.ToServer(srv)
	serverChanged := &model.ServerChangedEvent{
		Node:   s,
		Action: model.ServerActionStopped,
	}
	if err := r.serverSubscriptionService.Notify(serverChanged); err != nil {
		return nil, err
	}

	return &model.StopServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           s,
	}, nil
}
