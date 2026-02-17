package main

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/UnAfraid/wg-ui/pkg/api"
	"github.com/UnAfraid/wg-ui/pkg/auth"
	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/config"
	"github.com/UnAfraid/wg-ui/pkg/datastore"
	"github.com/UnAfraid/wg-ui/pkg/datastore/bbolt"
	"github.com/UnAfraid/wg-ui/pkg/dbx"
	"github.com/UnAfraid/wg-ui/pkg/manage"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/UnAfraid/wg-ui/pkg/subscription"
	"github.com/UnAfraid/wg-ui/pkg/user"
	"github.com/UnAfraid/wg-ui/pkg/wireguard"
	_ "github.com/UnAfraid/wg-ui/pkg/wireguard/darwin"         // Register darwin backend
	_ "github.com/UnAfraid/wg-ui/pkg/wireguard/linux"          // Register linux backend
	_ "github.com/UnAfraid/wg-ui/pkg/wireguard/networkmanager" // Register networkmanager backend
)

const (
	appName = "wg-ui"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339,
	})

	conf, err := config.Load(appName)
	if err != nil {
		logrus.
			WithError(err).
			Fatal("failed to initialize config")
		return
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGTERM, syscall.SIGINT)

	if _, err := maxprocs.Set(maxprocs.Logger(logrus.Printf)); err != nil {
		logrus.
			WithError(err).
			Error("failed to set maxprocs")
		return
	}

	debugServer := &http.Server{
		Addr: conf.DebugServer.Address(),
	}

	if conf.DebugServer.Enabled {
		go func() {
			logrus.WithField("address", conf.DebugServer.Address()).Info("Starting serving debug server")
			if err := debugServer.ListenAndServe(); err != nil {
				logrus.
					WithError(err).
					Fatal("Failed to serve debug")
				return
			}
		}()
	}

	logrus.Info("initializing database..")
	db, err := datastore.NewBBoltDB(conf.BoltDB.Path, conf.BoltDB.Timeout)
	if err != nil {
		logrus.
			WithError(err).
			Fatal("failed initialize datastore")
		return
	}

	jwtSecretBytes, err := base64.StdEncoding.DecodeString(conf.JwtSecret)
	if err != nil {
		logrus.
			WithError(err).
			Fatal("failed to base64 decode jwt secret")
		return
	}

	transactionScoper := dbx.NewBBoltTransactionScoper(db)
	subscriptionImpl := subscription.NewInMemorySubscription()

	serverRepository := bbolt.NewServerRepository(db)
	serverService := server.NewService(serverRepository, transactionScoper, subscriptionImpl)

	peerRepository := bbolt.NewPeerRepository(db)
	peerService := peer.NewService(peerRepository, transactionScoper, serverService, subscriptionImpl)

	userRepository := bbolt.NewUserRepository(db)
	userService, err := user.NewService(userRepository, transactionScoper, subscriptionImpl, conf.Initial.Email, conf.Initial.Password)
	if err != nil {
		logrus.
			WithError(err).
			Fatal("failed to initialize user service")
		return
	}

	backendRepository := bbolt.NewBackendRepository(db)
	serverCounter := backend.NewServerCounter(serverRepository)
	backendService := backend.NewService(backendRepository, serverCounter, transactionScoper, subscriptionImpl)

	wireguardRegistry := wireguard.NewRegistry()
	wireguardService := wireguard.NewService(wireguardRegistry)
	defer func() {
		if err := wireguardService.Close(context.Background()); err != nil {
			logrus.
				WithError(err).
				Error("failed to close wireguard service")
		}
	}()

	authService := auth.NewService(jwt.SigningMethodHS256, jwtSecretBytes, jwtSecretBytes, conf.JwtDuration)

	manageService := manage.NewService(
		transactionScoper,
		userService,
		backendService,
		serverService,
		peerService,
		wireguardService,
		conf.AutomaticStatsUpdateInterval,
		conf.AutomaticStatsUpdateOnlyWithSubscribers,
	)
	defer manageService.Close()

	router := api.NewRouter(
		conf,
		authService,
		userService,
		serverService,
		peerService,
		backendService,
		manageService,
	)

	httpServer := http.Server{
		Addr:    conf.HttpServer.Address(),
		Handler: router,
	}

	go func() {
		logrus.WithField("address", conf.HttpServer.Address()).Info("Starting serving http server")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.
				WithError(err).
				Fatal("failed to listen and serve http server")
		}
	}()

	<-shutdownChan
	logrus.Info("Shutting down")

	logrus.Info("Shutting down http server")
	httpServerShutdownTimeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(httpServerShutdownTimeoutCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.
			WithError(err).
			Fatal("failed to shutdown http server")
		return
	}

	if conf.DebugServer.Enabled {
		logrus.Info("Shutting down debug http server")
		debugHttpServerShutdownTimeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := debugServer.Shutdown(debugHttpServerShutdownTimeoutCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.
				WithError(err).
				Fatal("failed to shutdown debug server")
			return
		}
	}
}
