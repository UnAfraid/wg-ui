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

	"github.com/UnAfraid/wg-ui/api"
	"github.com/UnAfraid/wg-ui/api/subscription"
	"github.com/UnAfraid/wg-ui/config"
	"github.com/UnAfraid/wg-ui/datastore"
	"github.com/UnAfraid/wg-ui/datastore/bbolt"
	"github.com/UnAfraid/wg-ui/interfacestats"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/user"
	"github.com/UnAfraid/wg-ui/wg"
	"github.com/glendc/go-external-ip"
	"github.com/go-chi/jwtauth/v5"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"
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

	publicIp := conf.PublicIpAddress
	if len(publicIp) == 0 {
		logrus.Info("no public ip is configured, attempting to detect..")

		consensus := externalip.DefaultConsensus(nil, nil)
		if err := consensus.UseIPProtocol(4); err != nil {
			logrus.
				WithError(err).
				Fatal("failed to configure consensus to ipv4 only")
			return
		}

		externalIp, err := consensus.ExternalIP()
		if err != nil {
			logrus.
				WithError(err).
				Fatal("failed to get external ip")
			return
		}
		publicIp = externalIp.String()

		logrus.
			WithField("publicIp", publicIp).
			Info("public ip address detected")
	} else {
		logrus.
			WithField("publicIp", publicIp).
			Info("using configured public ip address")
	}

	subscriptionImpl := subscription.NewInMemorySubscription()
	nodeSubscriptionService := subscription.NewNodeService(subscriptionImpl)

	userRepository := bbolt.NewUserRepository(db)
	userService, err := user.NewService(userRepository, conf.Initial.Email, conf.Initial.Password)
	if err != nil {
		logrus.
			WithError(err).
			Fatal("failed to initialize user service")
		return
	}
	userSubscriptionService := subscription.NewUserService(subscriptionImpl)

	serverRepository := bbolt.NewServerRepository(db)
	serverService := server.NewService(serverRepository)
	serverSubscriptionService := subscription.NewServerService(subscriptionImpl)

	peerRepository := bbolt.NewPeerRepository(db)
	peerService := peer.NewService(peerRepository, serverService, publicIp)
	peerSubscriptionService := subscription.NewPeerService(subscriptionImpl)

	wgService, err := wg.NewService(serverService, peerService)
	if err != nil {
		logrus.
			WithError(err).
			Fatal("failed to initialize WireGuard service")
		return
	}
	defer wgService.Close()

	interfaceStatsService := interfacestats.NewService(wgService, serverService, serverSubscriptionService)
	defer interfaceStatsService.Close()

	router := api.NewRouter(
		conf,
		conf.CorsAllowedOrigins,
		jwtauth.New("HS256", jwtSecretBytes, nil),
		nodeSubscriptionService,
		userService,
		userSubscriptionService,
		serverService,
		serverSubscriptionService,
		peerService,
		peerSubscriptionService,
		wgService,
	)

	httpServer := http.Server{
		Addr:    conf.HttpServer.Address(),
		Handler: router,
	}

	go func() {
		logrus.WithField("address", conf.HttpServer.Address()).Info("Starting serving http server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
