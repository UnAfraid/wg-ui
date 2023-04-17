package api

import (
	"errors"
)

var (
	ErrNotImplemented         = errors.New("not implemented")
	ErrUserNotFound           = errors.New("user not found")
	ErrAuthenticationRequired = errors.New("authentication required")
)
