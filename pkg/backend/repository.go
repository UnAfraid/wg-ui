package backend

import "context"

type Repository interface {
	FindOne(ctx context.Context, options *FindOneOptions) (*Backend, error)
	FindAll(ctx context.Context, options *FindOptions) ([]*Backend, error)
	Create(ctx context.Context, backend *Backend) (*Backend, error)
	Update(ctx context.Context, backend *Backend, fieldMask *UpdateFieldMask) (*Backend, error)
	Delete(ctx context.Context, backendId string, deleteUserId string) (*Backend, error)
}
