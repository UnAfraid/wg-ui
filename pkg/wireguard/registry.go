package wireguard

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

// Registry manages active backend connections keyed by backend entity ID
type Registry struct {
	mu       sync.RWMutex
	backends map[string]*registryBackend
}

type registryBackend struct {
	instance    backend.Backend
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
func (r *Registry) Get(backendId string) backend.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.backends[backendId]
	if !ok {
		return nil
	}

	return entry.instance
}

// GetOrCreate retrieves an existing backend connection or creates a new one
func (r *Registry) GetOrCreate(ctx context.Context, backendId string, backendType string, rawURL string) (backend.Backend, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.backends[backendId]
	if ok && entry.backendType == backendType && entry.rawURL == rawURL {
		return entry.instance, nil
	}

	instance, err := backend.Create(backendType, rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend %s: %w", backendType, err)
	}

	r.backends[backendId] = &registryBackend{
		instance:    instance,
		backendType: backendType,
		rawURL:      rawURL,
	}

	if ok {
		if err := entry.instance.Close(ctx); err != nil {
			logrus.WithError(err).
				WithField("backendId", backendId).
				WithField("type", backendType).
				Warn("failed to close previous backend connection after url change")
		}

		logrus.WithField("backendId", backendId).
			WithField("type", backendType).
			WithField("url", rawURL).
			Info("recreated backend connection due to url change")
		return instance, nil
	}

	logrus.WithField("backendId", backendId).
		WithField("type", backendType).
		WithField("url", rawURL).
		Info("created backend connection")
	return instance, nil
}

// Remove removes and closes a backend connection
func (r *Registry) Remove(ctx context.Context, backendId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.backends[backendId]
	if !ok {
		return nil
	}
	delete(r.backends, backendId)

	// Close outside lock to avoid blocking
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
	// Copy backends and clear map under lock
	r.mu.Lock()
	defer r.mu.Unlock()

	backends := r.backends
	r.backends = make(map[string]*registryBackend)

	// Close all backends outside lock
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
