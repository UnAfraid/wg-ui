package api

import (
	"net/http"
	"time"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/apollotracing"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	gqlplayground "github.com/99designs/gqlgen/graphql/playground"
	"github.com/UnAfraid/wg-ui/api/exec"
	"github.com/UnAfraid/wg-ui/api/handler"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/UnAfraid/wg-ui/api/subscription"
	"github.com/UnAfraid/wg-ui/api/tools/frontend"
	"github.com/UnAfraid/wg-ui/api/tools/playground"
	"github.com/UnAfraid/wg-ui/api/tools/voyager"
	"github.com/UnAfraid/wg-ui/auth"
	"github.com/UnAfraid/wg-ui/config"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
)

const (
	dataLoaderWait     = 250 * time.Microsecond
	dataLoaderMaxBatch = 100
)

func NewRouter(
	conf *config.Config,
	corsAllowedOrigins []string,
	authService auth.Service,
	nodeSubscriptionService subscription.NodeService,
	userService user.Service,
	userSubscriptionService subscription.Service[*model.UserChangedEvent],
	serverService server.Service,
	serverSubscriptionService subscription.Service[*model.ServerChangedEvent],
	peerService peer.Service,
	peerSubscriptionService subscription.Service[*model.PeerChangedEvent],
	wgService wg.Service,
) http.Handler {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   corsAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	executableSchemaConfig := newConfig(
		authService,
		nodeSubscriptionService,
		userService,
		userSubscriptionService,
		serverService,
		serverSubscriptionService,
		peerService,
		peerSubscriptionService,
		wgService,
	)

	authHandler := handler.NewAuthenticationMiddleware(authService, userService)
	gqlHandler := gqlhandler.New(exec.NewExecutableSchema(executableSchemaConfig))
	gqlHandler.AddTransport(transport.Websocket{
		InitFunc:              authHandler.WebsocketMiddleware(),
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return checkOrigin(r, conf.SubscriptionAllowedOrigins)
			},
		},
	})
	gqlHandler.AddTransport(transport.Options{})
	gqlHandler.AddTransport(transport.GET{})
	gqlHandler.AddTransport(transport.POST{})
	gqlHandler.AddTransport(transport.MultipartForm{})

	gqlHandler.Use(extension.Introspection{})
	gqlHandler.Use(extension.FixedComplexityLimit(500))

	if conf.HttpServer.APQCacheEnabled {
		gqlHandler.Use(extension.AutomaticPersistedQuery{
			Cache: lru.New(1000),
		})
	}

	if conf.HttpServer.TracingEnabled {
		gqlHandler.Use(apollotracing.Tracer{})
	}

	router := chi.NewRouter()
	router.Use(corsMiddleware.Handler)

	router.Group(func(r chi.Router) {
		if conf.HttpServer.PlaygroundEnabled {
			r.Handle(conf.HttpServer.PlaygroundEndpoint, playground.Handler("GraphQL Playground", "/query"))
		}

		if conf.HttpServer.GraphiQLEnabled {
			r.Handle(conf.HttpServer.GraphiQLEndpoint, gqlplayground.Handler("GraphiQL Playground", "/query"))
		}

		if conf.HttpServer.SandboxExplorerEnabled {
			r.Handle(conf.HttpServer.SandboxExplorerEndpoint, gqlplayground.ApolloSandboxHandler("Apollo Sandbox Explorer", "/query"))
		}

		if conf.HttpServer.AltairEnabled {
			r.Handle(conf.HttpServer.AltairEndpoint, gqlplayground.AltairHandler("Altair Playground", "/query"))
		}

		if conf.HttpServer.VoyagerEnabled {
			r.Handle(conf.HttpServer.VoyagerEndpoint, voyager.Handler("Voyager", "/query"))
		}

		r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {})
	})

	router.Group(func(r chi.Router) {
		r.Use(authHandler.AuthenticationMiddleware())
		r.Use(handler.NewDataLoaderMiddleware(
			dataLoaderWait,
			dataLoaderMaxBatch,
			userService,
			serverService,
			peerService,
		))

		r.Handle("/query", gqlHandler)
		r.Handle("/api/query", gqlHandler)
	})

	if conf.HttpServer.FrontendEnabled {
		router.Mount("/", frontend.Handler())
	}

	return router
}
