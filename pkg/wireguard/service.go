package wireguard

import (
	"context"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

type Service interface {
	Device(ctx context.Context, name string) (*backend.Device, error)
	Up(ctx context.Context, options backend.ConfigureOptions) (*backend.Device, error)
	Down(ctx context.Context, name string) error
	Status(ctx context.Context, name string) (bool, error)
	Stats(ctx context.Context, name string) (*backend.InterfaceStats, error)
	PeerStats(ctx context.Context, name string, peerPublicKey string) (*backend.PeerStats, error)
	FindForeignServers(_ context.Context, knownInterfaces []string) ([]*backend.ForeignServer, error)
	Close(ctx context.Context) error
}

type service struct {
	backend backend.Backend
}

func NewService(backend backend.Backend) Service {
	return &service{
		backend: backend,
	}
}

func (s *service) Device(ctx context.Context, name string) (*backend.Device, error) {
	return s.backend.Device(ctx, name)
}

func (s *service) Up(ctx context.Context, options backend.ConfigureOptions) (*backend.Device, error) {
	return s.backend.Up(ctx, options)
}

func (s *service) Down(ctx context.Context, name string) error {
	return s.backend.Down(ctx, name)
}

func (s *service) Status(ctx context.Context, name string) (bool, error) {
	return s.backend.Status(ctx, name)
}

func (s *service) Stats(ctx context.Context, name string) (*backend.InterfaceStats, error) {
	return s.backend.Stats(ctx, name)
}

func (s *service) PeerStats(ctx context.Context, name string, peerPublicKey string) (*backend.PeerStats, error) {
	return s.backend.PeerStats(ctx, name, peerPublicKey)
}

func (s *service) FindForeignServers(ctx context.Context, knownInterfaces []string) ([]*backend.ForeignServer, error) {
	return s.backend.FindForeignServers(ctx, knownInterfaces)
}

func (s *service) Close(ctx context.Context) error {
	return s.backend.Close(ctx)
}
