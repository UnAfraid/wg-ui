package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/model"
)

type subscriptionResolver struct {
	*resolverRoot
}

func (r *subscriptionResolver) UserChanged(ctx context.Context) (<-chan *model.UserChangedEvent, error) {
	return r.userSubscriptionService.Subscribe(ctx)
}

func (r *subscriptionResolver) ServerChanged(ctx context.Context) (<-chan *model.ServerChangedEvent, error) {
	return r.serverSubscriptionService.Subscribe(ctx)
}

func (r *subscriptionResolver) PeerChanged(ctx context.Context) (<-chan *model.PeerChangedEvent, error) {
	return r.peerSubscriptionService.Subscribe(ctx)
}

func (r *subscriptionResolver) NodeChanged(ctx context.Context) (<-chan model.NodeChangedEvent, error) {
	return r.nodeSubscriptionService.Subscribe(ctx)
}
