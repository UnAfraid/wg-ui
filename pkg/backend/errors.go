package backend

import "errors"

var (
	ErrBackendNotFound                = errors.New("backend not found")
	ErrBackendIdAlreadyExists         = errors.New("backend id already exists")
	ErrBackendNameAlreadyInUse        = errors.New("backend name already in use")
	ErrBackendNotSupported            = errors.New("backend type not supported on this platform")
	ErrBackendHasServers              = errors.New("backend has servers and cannot be deleted")
	ErrInvalidBackendURL              = errors.New("invalid backend URL")
	ErrUnknownBackendType             = errors.New("unknown backend type")
	ErrCreateBackendOptionsRequired   = errors.New("create backend options required")
	ErrUpdateBackendOptionsRequired   = errors.New("update backend options required")
	ErrUpdateBackendFieldMaskRequired = errors.New("update backend field mask required")
)
