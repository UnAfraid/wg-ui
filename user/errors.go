package user

import (
	"errors"
)

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrIdRequired              = errors.New("id is required")
	ErrEmailRequired           = errors.New("email is required")
	ErrEmailInvalid            = errors.New("email is invalid")
	ErrOneOptionRequired       = errors.New("one option is required")
	ErrOnlyOneOptionAllowed    = errors.New("only one option is allowed")
	ErrUserNotFound            = errors.New("user not found")
	ErrUserIdAlreadyExists     = errors.New("user id already exists")
	ErrCreateOptionsRequired   = errors.New("create options are required")
	ErrUpdateOptionsRequired   = errors.New("update options are required")
	ErrUpdateFieldMaskRequired = errors.New("update field mask are required")
)
