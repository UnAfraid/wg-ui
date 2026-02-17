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
	backends map[string]backend.Backend
}

// NewRegistry creates a new connection registry
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]backend.Backend),
	}
}

// Get retrieves a backend connection by ID, returns nil if not found
func (r *Registry) Get(backendId string) backend.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends[backendId]
}

// GetOrCreate retrieves an existing backend connection or creates a new one
func (r *Registry) GetOrCreate(backendId string, backendType string) (backend.Backend, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.backends[backendId]; ok {
		return b, nil
	}

	b, err := backend.Create(backendType)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend %s: %w", backendType, err)
	}

	r.backends[backendId] = b
	logrus.WithField("backendId", backendId).WithField("type", backendType).Info("created backend connection")
	return b, nil
}

// Remove removes and closes a backend connection
func (r *Registry) Remove(ctx context.Context, backendId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.backends[backendId]
	if !ok {
		return nil
	}

	delete(r.backends, backendId)
	if err := b.Close(ctx); err != nil {
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
	defer r.mu.Unlock()

	var errs []error
	for id, b := range r.backends {
		if err := b.Close(ctx); err != nil {
			logrus.WithError(err).WithField("backendId", id).Error("failed to close backend")
			errs = append(errs, fmt.Errorf("backend %s: %w", id, err))
		}
	}
	r.backends = make(map[string]backend.Backend)

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
