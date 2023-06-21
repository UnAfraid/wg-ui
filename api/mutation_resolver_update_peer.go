package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/server"
)

func (r *mutationResolver) UpdatePeer(ctx context.Context, input model.UpdatePeerInput) (*model.UpdatePeerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	peerId, err := input.ID.String(model.IdKindPeer)
	if err != nil {
		return nil, err
	}

	updateOptions, updateFieldMask := model.UpdatePeerInputToUpdatePeerOptionsAndUpdatePeerFieldMask(input)
	peer, err := r.peerService.UpdatePeer(ctx, peerId, updateOptions, updateFieldMask, userId)
	if err != nil {
		return nil, err
	}

	err = r.withServer(ctx, peer.ServerId, func(svc *server.Server) {
		if svc.Enabled && svc.Running {
			err = r.wgService.UpdatePeer(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peer)
		}
	})
	if err != nil {
		return nil, err
	}

	p := model.ToPeer(peer)
	peerChanged := &model.PeerChangedEvent{
		Node:   p,
		Action: model.PeerActionUpdated,
	}
	if err := r.peerSubscriptionService.Notify(peerChanged); err != nil {
		return nil, err
	}

	return &model.UpdatePeerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Peer:             p,
	}, nil
}
