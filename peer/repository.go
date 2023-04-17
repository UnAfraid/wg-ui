package peer

import (
	"context"
)

type Repository interface {
	FindOne(ctx context.Context, options *FindOneOptions) (*Peer, error)
	FindAll(ctx context.Context, options *FindOptions) ([]*Peer, error)
	Create(ctx context.Context, peer *Peer) (*Peer, error)
	Update(ctx context.Context, peer *Peer, fieldMask *UpdateFieldMask) (*Peer, error)
	Delete(ctx context.Context, peerId string, deleteUserId string) (*Peer, error)
}
