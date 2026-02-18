package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/dbx"
	"github.com/UnAfraid/wg-ui/pkg/subscription"
)

var (
	subscriptionPath = path.Join("node", "Server")
)

type Service interface {
	FindServer(ctx context.Context, options *FindOneOptions) (*Server, error)
	FindServers(ctx context.Context, options *FindOptions) ([]*Server, error)
	CreateServer(ctx context.Context, options *CreateOptions, userId string) (*Server, error)
	UpdateServer(ctx context.Context, serverId string, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) (*Server, error)
	DeleteServer(ctx context.Context, serverId string, userId string) (*Server, error)
	Subscribe(ctx context.Context) (<-chan *ChangedEvent, error)
	HasSubscribers() bool
}

type service struct {
	serverRepository  Repository
	transactionScoper dbx.TransactionScoper
	subscription      subscription.Subscription
}

func NewService(serverRepository Repository, transactionScoper dbx.TransactionScoper, subscription subscription.Subscription) Service {
	return &service{
		serverRepository:  serverRepository,
		transactionScoper: transactionScoper,
		subscription:      subscription,
	}
}

func (s *service) FindServer(ctx context.Context, options *FindOneOptions) (*Server, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}
	return s.serverRepository.FindOne(ctx, options)
}

func (s *service) FindServers(ctx context.Context, options *FindOptions) ([]*Server, error) {
	return s.serverRepository.FindAll(ctx, options)
}

func (s *service) CreateServer(ctx context.Context, options *CreateOptions, userId string) (*Server, error) {
	server, err := processCreateServer(options, userId)
	if err != nil {
		return nil, err
	}

	if err := server.validate(nil); err != nil {
		return nil, err
	}

	if err := s.validateServerName(ctx, options.Name); err != nil {
		return nil, err
	}

	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*Server, error) {
		createdServer, err := s.serverRepository.Create(ctx, server)
		if err != nil {
			return nil, err
		}

		if err = s.notify(ChangedActionCreated, createdServer); err != nil {
			logrus.WithError(err).Warn("failed to notify server created event")
		}

		return createdServer, nil
	})
}

func (s *service) UpdateServer(ctx context.Context, serverId string, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) (*Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*Server, error) {
		server, err := s.findServerById(ctx, serverId)
		if err != nil {
			return nil, err
		}

		if err := processUpdateServer(server, options, fieldMask, userId); err != nil {
			return nil, err
		}

		if err := server.validate(fieldMask); err != nil {
			return nil, err
		}

		updatedServer, err := s.serverRepository.Update(ctx, server, fieldMask)
		if err != nil {
			return nil, err
		}

		action := ChangedActionUpdated
		if fieldMask.Running {
			if updatedServer.Running {
				action = ChangedActionStarted
			} else {
				action = ChangedActionStopped
			}
		} else if fieldMask.Stats {
			action = ChangedActionInterfaceStatsUpdated
		}

		if err = s.notify(action, updatedServer); err != nil {
			logrus.WithError(err).Warn("failed to notify server updated event")
		}

		return updatedServer, nil
	})
}

func (s *service) DeleteServer(ctx context.Context, serverId string, userId string) (*Server, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*Server, error) {
		server, err := s.findServerById(ctx, serverId)
		if err != nil {
			return nil, err
		}

		deletedServer, err := s.serverRepository.Delete(ctx, server.Id, userId)
		if err != nil {
			return nil, err
		}

		if err = s.notify(ChangedActionDeleted, deletedServer); err != nil {
			logrus.WithError(err).Warn("failed to notify server deleted event")
		}

		return deletedServer, nil
	})
}

func (s *service) findServerById(ctx context.Context, serverId string) (*Server, error) {
	server, err := s.serverRepository.FindOne(ctx, &FindOneOptions{
		IdOption: &IdOption{
			Id: serverId,
		},
	})
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}
	return server, nil
}

func (s *service) validateServerName(ctx context.Context, name string) error {
	existingServer, err := s.serverRepository.FindOne(ctx, &FindOneOptions{
		NameOption: &NameOption{
			Name: name,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find existing server by name: %s - %w", name, err)
	}
	if existingServer != nil {
		return ErrServerNameAlreadyInUse
	}
	return nil
}

func newId() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func processCreateServer(options *CreateOptions, userId string) (*Server, error) {
	if options == nil {
		return nil, ErrCreateServerOptionsRequired
	}

	var publicKey string
	if len(strings.TrimSpace(options.PrivateKey)) == 0 {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
		options.PrivateKey = key.String()
		publicKey = key.PublicKey().String()
	} else {
		key, err := wgtypes.ParseKey(options.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		publicKey = key.PublicKey().String()
	}

	if options.MTU == 0 {
		options.MTU = 1420
	}

	id, err := newId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new id: %w", err)
	}

	now := time.Now()

	return &Server{
		Id:           id,
		Name:         options.Name,
		Description:  options.Description,
		BackendId:    options.BackendId,
		Enabled:      options.Enabled,
		Running:      options.Running,
		PublicKey:    publicKey,
		PrivateKey:   options.PrivateKey,
		ListenPort:   options.ListenPort,
		FirewallMark: options.FirewallMark,
		Address:      options.Address,
		DNS:          options.DNS,
		MTU:          options.MTU,
		Hooks:        options.Hooks,
		CreateUserId: userId,
		CreatedAt:    now,
		UpdatedAt:    now,
		DeletedAt:    nil,
	}, nil
}

func processUpdateServer(server *Server, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) error {
	if options == nil {
		return ErrUpdateServerOptionsRequired
	}

	if fieldMask == nil {
		return ErrUpdateServerFieldMaskRequired
	}

	if fieldMask.PrivateKey {
		if len(strings.TrimSpace(options.PrivateKey)) == 0 {
			return errors.New("private key is required")
		}

		key, err := wgtypes.ParseKey(options.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		options.PrivateKey = key.String()
	}

	if userId != "" {
		server.UpdateUserId = userId
		fieldMask.UpdateUserId = true
	}

	if err := server.update(options, fieldMask); err != nil {
		return err
	}
	server.UpdatedAt = time.Now()
	return nil
}

func (s *service) notify(action string, server *Server) error {
	bytes, err := json.Marshal(ChangedEvent{Action: action, Server: server})
	if err != nil {
		return err
	}

	if err := s.subscription.Notify(bytes, path.Join(subscriptionPath, server.Id)); err != nil {
		return fmt.Errorf("failed to notify server changed event: %w", err)
	}
	return nil
}

func (s *service) Subscribe(ctx context.Context) (<-chan *ChangedEvent, error) {
	bytesChannel, err := s.subscription.Subscribe(ctx, path.Join(subscriptionPath, "*"))
	if err != nil {
		return nil, err
	}

	observerChan := make(chan *ChangedEvent)
	go func() {
		defer close(observerChan)

		for bytes := range bytesChannel {
			var changedEvent *ChangedEvent
			if err := json.Unmarshal(bytes, &changedEvent); err != nil {
				logrus.WithError(err).Warn("failed to decode server changed event")
				return
			}
			observerChan <- changedEvent
		}
	}()

	return observerChan, nil
}

func (s *service) HasSubscribers() bool {
	return s.subscription.HasSubscribers(path.Join(subscriptionPath, "*"))
}
