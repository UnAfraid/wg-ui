package api

import (
	"context"

	"github.com/UnAfraid/wg-ui/api/exec"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/api/subscription"
	"github.com/UnAfraid/wg-ui/auth"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
)

type resolverRoot struct {
	authService               auth.Service
	nodeSubscriptionService   subscription.NodeService
	userService               user.Service
	userSubscriptionService   subscription.Service[*model.UserChangedEvent]
	serverService             server.Service
	serverSubscriptionService subscription.Service[*model.ServerChangedEvent]
	peerService               peer.Service
	peerSubscriptionService   subscription.Service[*model.PeerChangedEvent]
	wgService                 wg.Service
}

func (r *resolverRoot) Peer() exec.PeerResolver {
	return &peerResolver{r}
}

func (r *resolverRoot) Server() exec.ServerResolver {
	return &serverResolver{r}
}

func (r *resolverRoot) User() exec.UserResolver {
	return &userResolver{r}
}

func (r *resolverRoot) Query() exec.QueryResolver {
	return &queryResolver{r}
}

func (r *resolverRoot) Mutation() exec.MutationResolver {
	return &mutationResolver{r}
}

func (r *resolverRoot) Subscription() exec.SubscriptionResolver {
	return &subscriptionResolver{r}
}

func (r *resolverRoot) withServer(ctx context.Context, serverId string, callback func(svc *server.Server)) error {
	svc, err := r.serverService.FindServer(ctx, &server.FindOneOptions{
		IdOption: &server.IdOption{
			Id: serverId,
		},
		NameOption: nil,
	})
	if err != nil {
		return err
	}
	if svc != nil {
		callback(svc)
	}
	return nil
}
