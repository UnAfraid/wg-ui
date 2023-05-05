package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/server"
)

func (r *mutationResolver) DeletePeer(ctx context.Context, input model.DeletePeerInput) (*model.DeletePeerPayload, error) {
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

	peer, err := r.peerService.DeletePeer(ctx, peerId, userId)
	if err != nil {
		return nil, err
	}

	err = r.withServer(ctx, peer.ServerId, func(svc *server.Server) {
		if svc.Enabled && svc.Running {
			err = r.wgService.RemovePeer(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peer)
		}
	})
	if err != nil {
		return nil, err
	}

	p := model.ToPeer(peer)
	peerChanged := &model.PeerChangedEvent{
		Node:   p,
		Action: model.PeerActionDeleted,
	}
	if err := r.peerSubscriptionService.Notify(peerChanged); err != nil {
		return nil, err
	}

	return &model.DeletePeerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Peer:             p,
	}, err
}
