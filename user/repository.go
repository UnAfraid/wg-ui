package user

import (
	"context"
)

type Repository interface {
	FindOne(ctx context.Context, options *FindOneOptions) (*User, error)
	FindAll(ctx context.Context, options *FindOptions) ([]*User, error)
	Create(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User, fieldMask *UpdateFieldMask) (*User, error)
	Delete(ctx context.Context, userId string) (*User, error)
}
