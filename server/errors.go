package server

import (
	"errors"
)

var (
	ErrIdRequired                    = errors.New("id is required")
	ErrNameRequired                  = errors.New("name is required")
	ErrInvalidName                   = errors.New("name is invalid")
	ErrOneOptionRequired             = errors.New("one option is required")
	ErrOnlyOneOptionAllowed          = errors.New("only one option is allowed")
	ErrServerNotFound                = errors.New("server not found")
	ErrServerIdAlreadyExists         = errors.New("server id already exists")
	ErrServerNameAlreadyInUse        = errors.New("name is already in use")
	ErrInvalidMtu                    = errors.New("invalid MTU must be between 1280 and 1500")
	ErrCreateServerOptionsRequired   = errors.New("create server options are required")
	ErrUpdateServerOptionsRequired   = errors.New("update server options are required")
	ErrUpdateServerFieldMaskRequired = errors.New("update server field mask are required")
)
