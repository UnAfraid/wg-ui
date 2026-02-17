package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/UnAfraid/wg-ui/pkg/dbx"
	"github.com/UnAfraid/wg-ui/pkg/subscription"
	wgbackend "github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

var (
	subscriptionPath = path.Join("node", "Backend")
)

type Service interface {
	FindBackend(ctx context.Context, options *FindOneOptions) (*Backend, error)
	FindBackends(ctx context.Context, options *FindOptions) ([]*Backend, error)
	CreateBackend(ctx context.Context, options *CreateOptions, userId string) (*Backend, error)
	UpdateBackend(ctx context.Context, backendId string, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) (*Backend, error)
	DeleteBackend(ctx context.Context, backendId string, userId string) (*Backend, error)
	RegisteredTypes(ctx context.Context) ([]string, error)
	Subscribe(ctx context.Context) (<-chan *ChangedEvent, error)
	HasSubscribers() bool
}

// ServerCounter checks if a backend has servers (to avoid circular dependency with server package)
type ServerCounter interface {
	CountServersByBackendId(ctx context.Context, backendId string) (int, error)
	CountEnabledServersByBackendId(ctx context.Context, backendId string) (int, error)
}

type service struct {
	backendRepository Repository
	serverCounter     ServerCounter
	transactionScoper dbx.TransactionScoper
	subscription      subscription.Subscription
}

func NewService(backendRepository Repository, serverCounter ServerCounter, transactionScoper dbx.TransactionScoper, subscription subscription.Subscription) Service {
	return &service{
		backendRepository: backendRepository,
		serverCounter:     serverCounter,
		transactionScoper: transactionScoper,
		subscription:      subscription,
	}
}

func (s *service) FindBackend(ctx context.Context, options *FindOneOptions) (*Backend, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}
	return s.backendRepository.FindOne(ctx, options)
}

func (s *service) FindBackends(ctx context.Context, options *FindOptions) ([]*Backend, error) {
	return s.backendRepository.FindAll(ctx, options)
}

func (s *service) CreateBackend(ctx context.Context, options *CreateOptions, userId string) (*Backend, error) {
	backend, err := processCreateBackend(options, userId)
	if err != nil {
		return nil, err
	}

	if err := backend.validate(nil); err != nil {
		return nil, err
	}

	// Validate that backend type is supported on this platform
	if !wgbackend.IsSupported(backend.Type()) {
		return nil, ErrBackendNotSupported
	}

	if err := s.validateBackendName(ctx, options.Name); err != nil {
		return nil, err
	}

	// Validate that no backend of this type already exists
	if err := s.validateBackendType(ctx, backend.Type()); err != nil {
		return nil, err
	}

	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*Backend, error) {
		createdBackend, err := s.backendRepository.Create(ctx, backend)
		if err != nil {
			return nil, err
		}

		if err = s.notify(ChangedActionCreated, createdBackend); err != nil {
			logrus.WithError(err).Warn("failed to notify backend created event")
		}

		return createdBackend, nil
	})
}

func (s *service) UpdateBackend(ctx context.Context, backendId string, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) (*Backend, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*Backend, error) {
		backend, err := s.findBackendById(ctx, backendId)
		if err != nil {
			return nil, err
		}

		originalType := backend.Type()
		originalName := backend.Name

		if err := processUpdateBackend(backend, options, fieldMask, userId); err != nil {
			return nil, err
		}

		if err := backend.validate(fieldMask); err != nil {
			return nil, err
		}

		if fieldMask.Name && backend.Name != originalName {
			if err := s.validateBackendName(ctx, backend.Name); err != nil {
				return nil, err
			}
		}

		// Prevent changing the backend URL scheme (type)
		if fieldMask.Url {
			newType := backend.Type()
			if newType != originalType {
				return nil, ErrBackendTypeChangeNotAllowed
			}
		}

		// Prevent disabling a backend that has enabled servers
		if fieldMask.Enabled && !backend.Enabled {
			enabledCount, err := s.serverCounter.CountEnabledServersByBackendId(ctx, backendId)
			if err != nil {
				return nil, fmt.Errorf("failed to check enabled servers for backend: %w", err)
			}
			if enabledCount > 0 {
				return nil, ErrBackendHasEnabledServers
			}
		}

		updatedBackend, err := s.backendRepository.Update(ctx, backend, fieldMask)
		if err != nil {
			return nil, err
		}

		action := ChangedActionUpdated
		if fieldMask.Enabled {
			if updatedBackend.Enabled {
				action = ChangedActionEnabled
			} else {
				action = ChangedActionDisabled
			}
		}

		if err = s.notify(action, updatedBackend); err != nil {
			logrus.WithError(err).Warn("failed to notify backend updated event")
		}

		return updatedBackend, nil
	})
}

func (s *service) DeleteBackend(ctx context.Context, backendId string, userId string) (*Backend, error) {
	return dbx.InTransactionScopeWithResult(ctx, s.transactionScoper, func(ctx context.Context) (*Backend, error) {
		backend, err := s.findBackendById(ctx, backendId)
		if err != nil {
			return nil, err
		}

		// Check if backend has servers
		serverCount, err := s.serverCounter.CountServersByBackendId(ctx, backendId)
		if err != nil {
			return nil, fmt.Errorf("failed to check servers for backend: %w", err)
		}
		if serverCount > 0 {
			return nil, ErrBackendHasServers
		}

		deletedBackend, err := s.backendRepository.Delete(ctx, backend.Id, userId)
		if err != nil {
			return nil, err
		}

		if err = s.notify(ChangedActionDeleted, deletedBackend); err != nil {
			logrus.WithError(err).Warn("failed to notify backend deleted event")
		}

		return deletedBackend, nil
	})
}

func (s *service) findBackendById(ctx context.Context, backendId string) (*Backend, error) {
	backend, err := s.backendRepository.FindOne(ctx, &FindOneOptions{
		IdOption: &IdOption{
			Id: backendId,
		},
	})
	if err != nil {
		return nil, err
	}
	if backend == nil {
		return nil, ErrBackendNotFound
	}
	return backend, nil
}

func (s *service) validateBackendName(ctx context.Context, name string) error {
	existingBackend, err := s.backendRepository.FindOne(ctx, &FindOneOptions{
		NameOption: &NameOption{
			Name: name,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find existing backend by name: %s - %w", name, err)
	}
	if existingBackend != nil {
		return ErrBackendNameAlreadyInUse
	}
	return nil
}

func (s *service) validateBackendType(ctx context.Context, backendType string) error {
	backends, err := s.backendRepository.FindAll(ctx, &FindOptions{
		Type: &backendType,
	})
	if err != nil {
		return fmt.Errorf("failed to find existing backends by type: %s - %w", backendType, err)
	}
	if len(backends) > 0 {
		return ErrBackendTypeAlreadyExists
	}
	return nil
}

func (s *service) RegisteredTypes(ctx context.Context) ([]string, error) {
	backends, err := s.backendRepository.FindAll(ctx, &FindOptions{})
	if err != nil {
		return nil, err
	}
	var types []string
	for _, b := range backends {
		t := b.Type()
		if !slices.Contains(types, t) {
			types = append(types, t)
		}
	}
	return types, nil
}

func newId() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func processCreateBackend(options *CreateOptions, userId string) (*Backend, error) {
	if options == nil {
		return nil, ErrCreateBackendOptionsRequired
	}

	// Validate URL format
	if _, err := ParseURL(options.Url); err != nil {
		return nil, err
	}

	id, err := newId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new id: %w", err)
	}

	now := time.Now()

	return &Backend{
		Id:           id,
		Name:         options.Name,
		Description:  options.Description,
		Url:          options.Url,
		Enabled:      options.Enabled,
		CreateUserId: userId,
		CreatedAt:    now,
		UpdatedAt:    now,
		DeletedAt:    nil,
	}, nil
}

func processUpdateBackend(backend *Backend, options *UpdateOptions, fieldMask *UpdateFieldMask, userId string) error {
	if options == nil {
		return ErrUpdateBackendOptionsRequired
	}

	if fieldMask == nil {
		return ErrUpdateBackendFieldMaskRequired
	}

	if fieldMask.Url {
		if len(strings.TrimSpace(options.Url)) == 0 {
			return fmt.Errorf("url is required")
		}

		if _, err := ParseURL(options.Url); err != nil {
			return err
		}
	}

	if userId != "" {
		backend.UpdateUserId = userId
		fieldMask.UpdateUserId = true
	}

	backend.update(options, fieldMask)
	backend.UpdatedAt = time.Now()
	return nil
}

func (s *service) notify(action string, backend *Backend) error {
	bytes, err := json.Marshal(ChangedEvent{Action: action, Backend: backend})
	if err != nil {
		return err
	}

	if err := s.subscription.Notify(bytes, path.Join(subscriptionPath, backend.Id)); err != nil {
		return fmt.Errorf("failed to notify backend changed event: %w", err)
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
				logrus.WithError(err).Warn("failed to decode backend changed event")
				continue
			}
			observerChan <- changedEvent
		}
	}()

	return observerChan, nil
}

func (s *service) HasSubscribers() bool {
	return s.subscription.HasSubscribers(path.Join(subscriptionPath, "*"))
}
