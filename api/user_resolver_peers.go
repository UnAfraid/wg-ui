package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
)

func (r *userResolver) Peers(ctx context.Context, u *model.User) ([]*model.Peer, error) {
	userId, err := u.ID.String(model.IdKindUser)
	if err != nil {
		return nil, err
	}

	peers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		CreateUserId: &userId,
	})
	if err != nil {
		return nil, err
	}

	return adapt.Array(peers, model.ToPeer), nil
}
