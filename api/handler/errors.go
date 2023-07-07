package handler

import (
	"errors"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrClaimsInvalid          = errors.New("claims are invalid")
)
