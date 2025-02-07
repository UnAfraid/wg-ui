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
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/handler"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/tools/frontend"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/tools/graphiqlsse"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/tools/playground"
	"github.com/UnAfraid/wg-ui/pkg/api/internal/tools/voyager"
	"github.com/UnAfraid/wg-ui/pkg/auth"
	"github.com/UnAfraid/wg-ui/pkg/config"
	"github.com/UnAfraid/wg-ui/pkg/manage"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/user"
)

const (
	websocketsKeepAlivePingInterval = 10 * time.Second
	dataLoaderWait                  = 250 * time.Microsecond
	dataLoaderMaxBatch              = 1000
)

func NewRouter(
	conf *config.Config,
	authService auth.Service,
	userService user.Service,
	serverService server.Service,
	peerService peer.Service,
	manageService manage.Service,
) http.Handler {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:      conf.CorsAllowedOrigins,
		AllowedMethods:      []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:      []string{"*"},
		AllowCredentials:    conf.CorsAllowCredentials,
		AllowPrivateNetwork: conf.CorsAllowPrivateNetwork,
	})

	executableSchemaConfig := newConfig(
		authService,
		userService,
		serverService,
		peerService,
		manageService,
	)

	authHandler := handler.NewAuthenticationMiddleware(authService, userService)
	gqlHandler := gqlhandler.New(resolver.NewExecutableSchema(executableSchemaConfig))
	gqlHandler.SetParserTokenLimit(15_000)
	gqlHandler.AddTransport(transport.Websocket{
		InitFunc:              authHandler.WebsocketMiddleware(),
		KeepAlivePingInterval: websocketsKeepAlivePingInterval,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return checkOrigin(r, conf.SubscriptionAllowedOrigins)
			},
		},
	})
	gqlHandler.AddTransport(transport.SSE{})
	gqlHandler.AddTransport(transport.Options{})
	gqlHandler.AddTransport(transport.GET{})
	gqlHandler.AddTransport(transport.POST{})
	gqlHandler.AddTransport(transport.MultipartForm{})
	gqlHandler.AddTransport(transport.UrlEncodedForm{})

	gqlHandler.Use(extension.Introspection{})
	gqlHandler.Use(extension.FixedComplexityLimit(500))

	if conf.HttpServer.APQCacheEnabled {
		gqlHandler.Use(extension.AutomaticPersistedQuery{
			Cache: lru.New[string](1000),
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
			handlerFunc := gqlplayground.Handler("GraphiQL Playground", "/query")
			switch conf.HttpServer.GraphiQLVersion {
			case "default":
			case "sse":
				handlerFunc = graphiqlsse.Handler("GraphiQL SSE Playground", "/query")
			default:
				logrus.WithField("graphiqlVersion", conf.HttpServer.GraphiQLVersion).Warn("unsupported graphiql version, using default")
			}
			r.Handle(conf.HttpServer.GraphiQLEndpoint, handlerFunc)
		}

		if conf.HttpServer.SandboxExplorerEnabled {
			r.Handle(conf.HttpServer.SandboxExplorerEndpoint, gqlplayground.ApolloSandboxHandler("Apollo Sandbox Explorer", "/query"))
		}

		if conf.HttpServer.AltairEnabled {
			r.Handle(conf.HttpServer.AltairEndpoint, gqlplayground.AltairHandler("Altair Playground", "/query", nil))
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

	if conf.HttpServer.FrontendEnabled && frontend.HasContent() {
		router.Mount("/", frontend.Handler())
	}

	return router
}
