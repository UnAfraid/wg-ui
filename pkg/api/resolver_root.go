package api

import (
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
)

type resolverRoot struct {
	queryResolver        resolver.QueryResolver
	mutationResolver     resolver.MutationResolver
	subscriptionResolver resolver.SubscriptionResolver
	userResolver         resolver.UserResolver
	serverResolver       resolver.ServerResolver
	peerResolver         resolver.PeerResolver
}

func (r *resolverRoot) Query() resolver.QueryResolver {
	return r.queryResolver
}

func (r *resolverRoot) Mutation() resolver.MutationResolver {
	return r.mutationResolver
}

func (r *resolverRoot) Subscription() resolver.SubscriptionResolver {
	return r.subscriptionResolver
}

func (r *resolverRoot) User() resolver.UserResolver {
	return r.userResolver
}

func (r *resolverRoot) Server() resolver.ServerResolver {
	return r.serverResolver
}

func (r *resolverRoot) Peer() resolver.PeerResolver {
	return r.peerResolver
}
