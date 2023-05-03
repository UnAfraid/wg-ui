package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/server"
)

func (r *mutationResolver) CreatePeer(ctx context.Context, input model.CreatePeerInput) (*model.CreatePeerPayload, error) {
	user, err := model.ContextToUser(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := user.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	serverId, err := input.ServerID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	peer, err := r.peerService.CreatePeer(ctx, serverId, model.CreatePeerInputToCreateOptions(input), userId)
	if err != nil {
		return nil, err
	}

	err = r.withServer(ctx, peer.ServerId, func(svc *server.Server) {
		if svc.Enabled && svc.Running {
			err = r.wgService.AddPeer(svc.Name, svc.PrivateKey, svc.ListenPort, svc.FirewallMark, peer)
		}
	})
	if err != nil {
		return nil, err
	}

	p := model.ToPeer(peer)
	peerChanged := &model.PeerChangedEvent{
		Node:   p,
		Action: model.PeerActionCreated,
	}
	if err := r.peerSubscriptionService.Notify(peerChanged); err != nil {
		return nil, err
	}

	return &model.CreatePeerPayload{
		ClientMutationID: input.ClientMutationID.Value(),
		Peer:             p,
	}, err
}
