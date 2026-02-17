package wireguard

import (
	"context"
	"errors"
	"fmt"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

// BackendRef is an interface for referencing a backend entity.
// This is implemented by pkg/backend.Backend to avoid circular imports.
type BackendRef interface {
	ID() string
	Type() string
	URL() string
}

type Service interface {
	Device(ctx context.Context, b BackendRef, name string) (*driver.Device, error)
	Up(ctx context.Context, b BackendRef, options driver.ConfigureOptions) (*driver.Device, error)
	Down(ctx context.Context, b BackendRef, name string) error
	Status(ctx context.Context, b BackendRef, name string) (bool, error)
	Stats(ctx context.Context, b BackendRef, name string) (*driver.InterfaceStats, error)
	PeerStats(ctx context.Context, b BackendRef, name string, peerPublicKey string) (*driver.PeerStats, error)
	FindForeignServers(ctx context.Context, b BackendRef, knownInterfaces []string) ([]*driver.ForeignServer, error)
	RemoveBackend(ctx context.Context, backendId string) error
	Close(ctx context.Context) error
}

type service struct {
	registry *Registry
}

func NewService(registry *Registry) Service {
	return &service{
		registry: registry,
	}
}

func (s *service) getBackend(ctx context.Context, b BackendRef) (driver.Backend, error) {
	return s.registry.GetOrCreate(ctx, b.ID(), b.Type(), b.URL())
}

func (s *service) Device(ctx context.Context, ref BackendRef, name string) (*driver.Device, error) {
	return withBackendRetry(ctx, s, ref, func(instance driver.Backend) (*driver.Device, error) {
		return instance.Device(ctx, name)
	})
}

func (s *service) Up(ctx context.Context, ref BackendRef, options driver.ConfigureOptions) (*driver.Device, error) {
	return withBackendRetry(ctx, s, ref, func(instance driver.Backend) (*driver.Device, error) {
		return instance.Up(ctx, options)
	})
}

func (s *service) Down(ctx context.Context, ref BackendRef, name string) error {
	_, err := withBackendRetry(ctx, s, ref, func(instance driver.Backend) (struct{}, error) {
		return struct{}{}, instance.Down(ctx, name)
	})
	return err
}

func (s *service) Status(ctx context.Context, ref BackendRef, name string) (bool, error) {
	return withBackendRetry(ctx, s, ref, func(instance driver.Backend) (bool, error) {
		return instance.Status(ctx, name)
	})
}

func (s *service) Stats(ctx context.Context, ref BackendRef, name string) (*driver.InterfaceStats, error) {
	return withBackendRetry(ctx, s, ref, func(instance driver.Backend) (*driver.InterfaceStats, error) {
		return instance.Stats(ctx, name)
	})
}

func (s *service) PeerStats(ctx context.Context, ref BackendRef, name string, peerPublicKey string) (*driver.PeerStats, error) {
	return withBackendRetry(ctx, s, ref, func(instance driver.Backend) (*driver.PeerStats, error) {
		return instance.PeerStats(ctx, name, peerPublicKey)
	})
}

func (s *service) FindForeignServers(ctx context.Context, ref BackendRef, knownInterfaces []string) ([]*driver.ForeignServer, error) {
	servers, err := withBackendRetry(ctx, s, ref, func(instance driver.Backend) ([]*driver.ForeignServer, error) {
		return instance.FindForeignServers(ctx, knownInterfaces)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}
	// Set BackendId on each foreign server
	for _, srv := range servers {
		srv.BackendId = ref.ID()
	}
	return servers, nil
}

func (s *service) RemoveBackend(ctx context.Context, backendId string) error {
	return s.registry.Remove(ctx, backendId)
}

func (s *service) Close(ctx context.Context) error {
	return s.registry.CloseAll(ctx)
}

func withBackendRetry[T any](
	ctx context.Context,
	s *service,
	ref BackendRef,
	operation func(instance driver.Backend) (T, error),
) (T, error) {
	var zero T

	instance, err := s.getBackend(ctx, ref)
	if err != nil {
		return zero, err
	}

	result, err := operation(instance)
	if err == nil || !errors.Is(err, driver.ErrConnectionStale) {
		return result, err
	}

	if removeErr := s.registry.Remove(ctx, ref.ID()); removeErr != nil {
		return zero, errors.Join(err, fmt.Errorf("failed to recreate stale backend instance %s: %w", ref.ID(), removeErr))
	}

	instance, err = s.getBackend(ctx, ref)
	if err != nil {
		return zero, err
	}

	return operation(instance)
}
