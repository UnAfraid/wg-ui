package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
)

func (r *queryResolver) Peers(ctx context.Context, query *string) ([]*model.Peer, error) {
	servers, err := r.peerService.FindPeers(ctx, &peer.FindOptions{
		Ids:   nil,
		Query: adapt.Dereference(query),
	})
	if err != nil {
		return nil, err
	}
	return adapt.Array(servers, model.ToPeer), nil
}
