package api

import (
	"github.com/UnAfraid/wg-ui/api/internal/mutation"
	peerResolver "github.com/UnAfraid/wg-ui/api/internal/peer"
	"github.com/UnAfraid/wg-ui/api/internal/query"
	"github.com/UnAfraid/wg-ui/api/internal/resolver"
	serverResolver "github.com/UnAfraid/wg-ui/api/internal/server"
	sybscriptionResolver "github.com/UnAfraid/wg-ui/api/internal/subscription"
	userResolver "github.com/UnAfraid/wg-ui/api/internal/user"
	"github.com/UnAfraid/wg-ui/auth"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
)

//go:generate go run github.com/99designs/gqlgen --config ../../gqlgen.yml generate
func newConfig(
	authService auth.Service,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	wgService wg.Service,
) resolver.Config {
	return resolver.Config{
		Resolvers: &resolverRoot{
			queryResolver: query.NewQueryResolver(
				wgService,
				peerService,
				serverService,
				userService,
			),
			mutationResolver: mutation.NewMutationResolver(
				authService,
				userService,
				serverService,
				peerService,
				wgService,
			),
			subscriptionResolver: sybscriptionResolver.NewSubscriptionResolver(
				userService,
				serverService,
				peerService,
			),
			userResolver: userResolver.NewUserResolver(
				serverService,
				peerService,
			),
			serverResolver: serverResolver.NewServerResolver(
				serverService,
				peerService,
				wgService,
			),
			peerResolver: peerResolver.NewPeerResolver(
				peerService,
				wgService,
			),
		},
		Directives: resolver.DirectiveRoot{
			Authenticated: authenticated,
		},
	}
}
