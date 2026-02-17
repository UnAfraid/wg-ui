package wireguard

import (
	"context"
	"fmt"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

// BackendRef is an interface for referencing a backend entity.
// This is implemented by pkg/backend.Backend to avoid circular imports.
type BackendRef interface {
	GetId() string
	Type() string
	GetURL() string
}

type Service interface {
	Device(ctx context.Context, b BackendRef, name string) (*backend.Device, error)
	Up(ctx context.Context, b BackendRef, options backend.ConfigureOptions) (*backend.Device, error)
	Down(ctx context.Context, b BackendRef, name string) error
	Status(ctx context.Context, b BackendRef, name string) (bool, error)
	Stats(ctx context.Context, b BackendRef, name string) (*backend.InterfaceStats, error)
	PeerStats(ctx context.Context, b BackendRef, name string, peerPublicKey string) (*backend.PeerStats, error)
	FindForeignServers(ctx context.Context, b BackendRef, knownInterfaces []string) ([]*backend.ForeignServer, error)
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

func (s *service) getBackend(ctx context.Context, b BackendRef) (backend.Backend, error) {
	return s.registry.GetOrCreate(ctx, b.GetId(), b.Type(), b.GetURL())
}

func (s *service) Device(ctx context.Context, ref BackendRef, name string) (*backend.Device, error) {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.Device(ctx, name)
}

func (s *service) Up(ctx context.Context, ref BackendRef, options backend.ConfigureOptions) (*backend.Device, error) {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.Up(ctx, options)
}

func (s *service) Down(ctx context.Context, ref BackendRef, name string) error {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return err
	}
	return b.Down(ctx, name)
}

func (s *service) Status(ctx context.Context, ref BackendRef, name string) (bool, error) {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return false, err
	}
	return b.Status(ctx, name)
}

func (s *service) Stats(ctx context.Context, ref BackendRef, name string) (*backend.InterfaceStats, error) {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.Stats(ctx, name)
}

func (s *service) PeerStats(ctx context.Context, ref BackendRef, name string, peerPublicKey string) (*backend.PeerStats, error) {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return nil, err
	}
	return b.PeerStats(ctx, name, peerPublicKey)
}

func (s *service) FindForeignServers(ctx context.Context, ref BackendRef, knownInterfaces []string) ([]*backend.ForeignServer, error) {
	b, err := s.getBackend(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}
	servers, err := b.FindForeignServers(ctx, knownInterfaces)
	if err != nil {
		return nil, err
	}
	// Set BackendId on each foreign server
	for _, srv := range servers {
		srv.BackendId = ref.GetId()
	}
	return servers, nil
}

func (s *service) RemoveBackend(ctx context.Context, backendId string) error {
	return s.registry.Remove(ctx, backendId)
}

func (s *service) Close(ctx context.Context) error {
	return s.registry.CloseAll(ctx)
}
