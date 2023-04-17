package api

import (
	"time"

	"github.com/UnAfraid/wg-ui/api/exec"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/api/subscription"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
	"github.com/go-chi/jwtauth/v5"
)

//go:generate go run github.com/99designs/gqlgen --config ../../gqlgen.yml generate
func newConfig(
	jwtDuration time.Duration,
	jwtAuth *jwtauth.JWTAuth,
	nodeSubscriptionService subscription.NodeService,
	userService user.Service,
	userSubscriptionService subscription.Service[*model.UserChangedEvent],
	serverService server.Service,
	serverSubscriptionService subscription.Service[*model.ServerChangedEvent],
	peerService peer.Service,
	peerSubscriptionService subscription.Service[*model.PeerChangedEvent],
	wgService wg.Service,
) exec.Config {
	authenticated := &authenticated{
		userService: userService,
	}
	return exec.Config{
		Resolvers: &resolverRoot{
			jwtDuration:               jwtDuration,
			jwtAuth:                   jwtAuth,
			nodeSubscriptionService:   nodeSubscriptionService,
			userService:               userService,
			userSubscriptionService:   userSubscriptionService,
			serverService:             serverService,
			serverSubscriptionService: serverSubscriptionService,
			peerService:               peerService,
			peerSubscriptionService:   peerSubscriptionService,
			wgService:                 wgService,
		},
		Directives: exec.DirectiveRoot{
			Authenticated: authenticated.directive,
		},
	}
}
