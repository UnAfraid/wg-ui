package wireguard

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

// Registry manages active backend connections keyed by backend entity ID
type Registry struct {
	mu          sync.RWMutex
	backends    map[string]*registryBackend
	createGroup singleflight.Group
}

type registryBackend struct {
	instance    driver.Backend
	backendType string
	rawURL      string
}

// NewRegistry creates a new connection registry
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]*registryBackend),
	}
}

// Get retrieves a backend connection by ID, returns nil if not found
func (r *Registry) Get(backendId string) driver.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.backends[backendId]
	if !ok {
		return nil
	}

	return entry.instance
}

// GetOrCreate retrieves an existing backend connection or creates a new one
func (r *Registry) GetOrCreate(ctx context.Context, backendId string, backendType string, rawURL string) (driver.Backend, error) {
	r.mu.RLock()
	entry, ok := r.backends[backendId]
	if ok && entry.backendType == backendType && entry.rawURL == rawURL {
		instance := entry.instance
		r.mu.RUnlock()
		return instance, nil
	}
	r.mu.RUnlock()

	createKey := fmt.Sprintf("%s\x00%s\x00%s", backendId, backendType, rawURL)
	value, err, _ := r.createGroup.Do(createKey, func() (interface{}, error) {
		r.mu.RLock()
		entry, ok := r.backends[backendId]
		if ok && entry.backendType == backendType && entry.rawURL == rawURL {
			instance := entry.instance
			r.mu.RUnlock()
			return instance, nil
		}
		r.mu.RUnlock()

		instance, err := driver.Create(ctx, backendType, rawURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create backend %s: %w", backendType, err)
		}

		var previous *registryBackend
		var recreate bool

		r.mu.Lock()
		entry, ok = r.backends[backendId]
		if ok && entry.backendType == backendType && entry.rawURL == rawURL {
			existing := entry.instance
			r.mu.Unlock()

			if closeErr := instance.Close(ctx); closeErr != nil {
				logrus.WithError(closeErr).
					WithField("backendId", backendId).
					WithField("type", backendType).
					WithField("url", rawURL).
					Warn("failed to close redundant backend connection")
			}

			return existing, nil
		}

		if ok {
			previous = entry
			recreate = true
		}

		r.backends[backendId] = &registryBackend{
			instance:    instance,
			backendType: backendType,
			rawURL:      rawURL,
		}
		r.mu.Unlock()

		if previous != nil {
			if err := previous.instance.Close(ctx); err != nil {
				logrus.WithError(err).
					WithField("backendId", backendId).
					WithField("type", backendType).
					Warn("failed to close previous backend connection after url change")
			}
		}

		logMessage := "created backend connection"
		if recreate {
			logMessage = "recreated backend connection due to url change"
		}

		logrus.WithField("backendId", backendId).
			WithField("type", backendType).
			WithField("url", rawURL).
			Info(logMessage)
		return instance, nil
	})
	if err != nil {
		return nil, err
	}

	instance, ok := value.(driver.Backend)
	if !ok || instance == nil {
		return nil, fmt.Errorf("failed to create backend %s: invalid backend instance", backendType)
	}
	return instance, nil
}

// Remove removes and closes a backend connection
func (r *Registry) Remove(ctx context.Context, backendId string) error {
	r.mu.Lock()
	entry, ok := r.backends[backendId]
	if !ok {
		r.mu.Unlock()
		return nil
	}
	delete(r.backends, backendId)
	r.mu.Unlock()

	if err := entry.instance.Close(ctx); err != nil {
		return fmt.Errorf("failed to close backend %s: %w", backendId, err)
	}

	logrus.WithField("backendId", backendId).Info("removed backend connection")
	return nil
}

// Has checks if a backend connection exists
func (r *Registry) Has(backendId string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.backends[backendId]
	return ok
}

// CloseAll closes all backend connections
func (r *Registry) CloseAll(ctx context.Context) error {
	r.mu.Lock()
	backends := r.backends
	r.backends = make(map[string]*registryBackend)
	r.mu.Unlock()

	var errs []error
	for id, entry := range backends {
		if err := entry.instance.Close(ctx); err != nil {
			logrus.WithError(err).WithField("backendId", id).Error("failed to close backend")
			errs = append(errs, fmt.Errorf("backend %s: %w", id, err))
		}
	}

	return errors.Join(errs...)
}

// List returns all backend IDs
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.backends))
	for id := range r.backends {
		ids = append(ids, id)
	}
	return ids
}
