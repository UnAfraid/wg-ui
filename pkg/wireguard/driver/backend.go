package driver

import "context"

type Backend interface {
	Device(ctx context.Context, name string) (*Device, error)
	Up(ctx context.Context, options ConfigureOptions) (*Device, error)
	Down(ctx context.Context, name string) error
	Status(ctx context.Context, name string) (bool, error)
	Stats(ctx context.Context, name string) (*InterfaceStats, error)
	PeerStats(ctx context.Context, name string, peerPublicKey string) (*PeerStats, error)
	FindForeignServers(ctx context.Context, knownInterfaces []string) ([]*ForeignServer, error)
	Close(ctx context.Context) error
}
