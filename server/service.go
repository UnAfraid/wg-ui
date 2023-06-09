package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Service interface {
	FindServer(ctx context.Context, options *FindOneOptions) (*Server, error)
	FindServers(ctx context.Context, options *FindOptions) ([]*Server, error)
	CreateServer(ctx context.Context, options *CreateOptions, userId string) (*Server, error)
	UpdateServer(ctx context.Context, serverId string, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) (*Server, error)
	DeleteServer(ctx context.Context, serverId string, userId string) (*Server, error)
}

type service struct {
	serverRepository Repository
}

func NewService(serverRepository Repository) Service {
	return &service{
		serverRepository: serverRepository,
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

	createdServer, err := s.serverRepository.Create(ctx, server)
	if err != nil {
		return nil, err
	}

	if err := createdServer.RunHooks(HookActionCreate); err != nil {
		logrus.
			WithError(err).
			WithField("server", createdServer.Name).
			Error("failed to run hooks on server create")
	}

	return createdServer, nil
}

func (s *service) UpdateServer(ctx context.Context, serverId string, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) (*Server, error) {
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

	if err := updatedServer.RunHooks(HookActionUpdate); err != nil {
		logrus.
			WithError(err).
			WithField("server", updatedServer.Name).
			Error("failed to run hooks on server update")
	}

	return updatedServer, nil
}

func (s *service) DeleteServer(ctx context.Context, serverId string, userId string) (*Server, error) {
	server, err := s.findServerById(ctx, serverId)
	if err != nil {
		return nil, err
	}

	deletedServer, err := s.serverRepository.Delete(ctx, server.Id, userId)
	if err != nil {
		return nil, err
	}

	if err := deletedServer.RunHooks(HookActionDelete); err != nil {
		logrus.
			WithError(err).
			WithField("server", deletedServer.Name).
			Error("failed to run hooks on server delete")
	}

	return deletedServer, nil
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

	if len(strings.TrimSpace(options.PrivateKey)) == 0 {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
		options.PrivateKey = key.String()
		options.PublicKey = key.PublicKey().String()
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
		Enabled:      options.Enabled,
		PublicKey:    options.PublicKey,
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
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return fmt.Errorf("failed to generate private key: %w", err)
			}
			options.PrivateKey = key.String()
			options.PublicKey = key.PublicKey().String()
		}
	}

	if userId != "" {
		server.UpdateUserId = userId
		fieldMask.UpdateUserId = true
	}

	server.update(options, fieldMask)
	server.UpdatedAt = time.Now()
	return nil
}
