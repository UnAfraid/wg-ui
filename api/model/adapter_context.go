package model

import (
	"context"
	"errors"
)

var (
	userCtxKey = &struct {
		name string
	}{"user"}
	userErrorCtxKey = &struct {
		name string
	}{"userError"}
	ErrUserNotFound = errors.New("user not found")
)

func ContextToUser(ctx context.Context) (*User, error) {
	if err, ok := ctx.Value(userErrorCtxKey).(error); ok {
		return nil, err
	}
	if u, ok := ctx.Value(userCtxKey).(*User); ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

func UserToContext(ctx context.Context, user *User, err error) context.Context {
	if err != nil {
		ctx = context.WithValue(ctx, userErrorCtxKey, err)
	}
	if user != nil {
		ctx = context.WithValue(ctx, userCtxKey, user)
	}
	return ctx
}
