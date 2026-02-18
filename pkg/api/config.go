package api

import (
	backendResolver "github.com/UnAfraid/wg-ui/pkg/api/internal/backend"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/directive"
	foreignResolver "github.com/UnAfraid/wg-ui/pkg/api/internal/foreign"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/mutation"
	peerResolver "github.com/UnAfraid/wg-ui/pkg/api/internal/peer"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/query"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	serverResolver "github.com/UnAfraid/wg-ui/pkg/api/internal/server"
	sybscriptionResolver "github.com/UnAfraid/wg-ui/pkg/api/internal/subscription"
	userResolver "github.com/UnAfraid/wg-ui/pkg/api/internal/user"
	"github.com/UnAfraid/wg-ui/pkg/auth"
	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/manage"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
)

//go:generate go tool github.com/99designs/gqlgen --config ../../gqlgen.yml generate
func newConfig(
	authService auth.Service,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	backendService backend.Service,
	manageService manage.Service,
) resolver.Config {
	return resolver.Config{
		Resolvers: &resolverRoot{
			queryResolver: query.NewQueryResolver(
				peerService,
				serverService,
				userService,
				backendService,
				manageService,
			),
			mutationResolver: mutation.NewMutationResolver(
				authService,
				manageService,
			),
			subscriptionResolver: sybscriptionResolver.NewSubscriptionResolver(
				userService,
				serverService,
				peerService,
				backendService,
			),
			userResolver: userResolver.NewUserResolver(
				serverService,
				peerService,
			),
			serverResolver: serverResolver.NewServerResolver(
				peerService,
			),
			peerResolver: peerResolver.NewPeerResolver(
				manageService,
			),
			backendResolver: backendResolver.NewBackendResolver(
				backendService,
				serverService,
				peerService,
				manageService,
			),
			foreignServerResolver: foreignResolver.NewForeignServerResolver(),
		},
		Directives: directive.NewDirectiveRoot(),
	}
}
