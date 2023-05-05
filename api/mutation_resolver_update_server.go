package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/hashicorp/go-multierror"
)

func (r *mutationResolver) UpdateServer(ctx context.Context, input model.UpdateServerInput) (*model.UpdateServerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	updateOptions, updateFieldMask, err := model.UpdateServerInputToUpdateOptionsAndUpdateFieldMask(input)
	if err != nil {
		return nil, err
	}

	serverId, err := input.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	updatedServer, err := r.serverService.UpdateServer(ctx, serverId, updateOptions, updateFieldMask, userId)
	if err != nil {
		return nil, err
	}

	var retErr error
	if updatedServer.Enabled && updateOptions.Running {
		peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
			ServerId: &serverId,
		})
		if err != nil {
			retErr = err
		} else {
			if err := r.wgService.ConfigureWireGuard(updatedServer.Name, updatedServer.PrivateKey, updatedServer.ListenPort, updatedServer.FirewallMark, peers); err != nil {
				if retErr != nil {
					retErr = multierror.Append(retErr, err)
				}
			}
		}
	}

	s := model.ToServer(updatedServer)
	serverChanged := &model.ServerChangedEvent{
		Node:   s,
		Action: model.ServerActionUpdated,
	}
	if err := r.serverSubscriptionService.Notify(serverChanged); err != nil {
		return nil, err
	}

	return &model.UpdateServerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Server:           s,
	}, retErr
}
