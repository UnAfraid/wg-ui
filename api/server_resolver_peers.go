package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
)

func (r *serverResolver) Peers(ctx context.Context, svc *model.Server) ([]*model.Peer, error) {
	serverId, err := svc.ID.String(model.IdKindServer)
	if err != nil {
		return nil, err
	}

	peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		ServerId: &serverId,
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(peers, model.ToPeer), nil
}
