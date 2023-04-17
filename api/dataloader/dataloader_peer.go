package dataloader

import (
	"context"
	"errors"

	"github.com/UnAfraid/dataloaden/v2/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
)

//go:generate go run github.com/UnAfraid/dataloaden/v2 -name PeerLoader -keyType string -valueType *github.com/UnAfraid/wg-ui/api/model.Peer
var peerLoaderCtxKey = &contextKey{"peerLoader"}

func PeerLoaderFromContext(ctx context.Context) (dataloader.DataLoader[string, *model.Peer], error) {
	dataLoader, ok := ctx.Value(peerLoaderCtxKey).(dataloader.DataLoader[string, *model.Peer])
	if !ok {
		return nil, errors.New("peer loader not found")
	}
	return dataLoader, nil
}

func peerFetcher(ctx context.Context, peerService peer.Service) func([]string) ([]*model.Peer, []error) {
	return func(ids []string) ([]*model.Peer, []error) {
		peers, err := peerService.FindPeers(ctx, &peer.FindOptions{
			Ids: ids,
		})
		if err != nil {
			return nil, repeatError(err, len(ids))
		}
		return adapt.Array(peers, model.ToPeer), nil
	}
}
