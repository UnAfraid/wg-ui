package model

import (
	"context"
	"errors"
)

var (
	userCtxKey = &struct {
		name string
	}{"user"}
	ErrUserNotFound = errors.New("user not found")
)

func ContextToUser(ctx context.Context) (*User, error) {
	u, ok := ctx.Value(userCtxKey).(*User)
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func UserToContext(ctx context.Context, user *User) (context.Context, error) {
	return context.WithValue(ctx, userCtxKey, user), nil
}
