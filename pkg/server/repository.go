package server

import (
	"context"
)

type Repository interface {
	FindOne(ctx context.Context, options *FindOneOptions) (*Server, error)
	FindAll(ctx context.Context, options *FindOptions) ([]*Server, error)
	Create(ctx context.Context, server *Server) (*Server, error)
	Update(ctx context.Context, server *Server, fieldMask *UpdateFieldMask) (*Server, error)
	Delete(ctx context.Context, serverId string, deleteUserId string) (*Server, error)
	CountByBackendId(ctx context.Context, backendId string) (int, error)
	CountEnabledByBackendId(ctx context.Context, backendId string) (int, error)
}
