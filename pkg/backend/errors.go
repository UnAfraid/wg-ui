package backend

import "errors"

var (
	ErrBackendNotFound                = errors.New("backend not found")
	ErrBackendIdAlreadyExists         = errors.New("backend id already exists")
	ErrBackendNameAlreadyInUse        = errors.New("backend name already in use")
	ErrBackendTypeAlreadyExists       = errors.New("backend of this type already exists")
	ErrBackendTypeChangeNotAllowed    = errors.New("backend type cannot be changed")
	ErrBackendNotSupported            = errors.New("backend type not supported on this platform")
	ErrBackendHasServers              = errors.New("backend has servers and cannot be deleted")
	ErrBackendHasEnabledServers       = errors.New("backend has enabled servers and cannot be disabled")
	ErrInvalidBackendURL              = errors.New("invalid backend URL")
	ErrUnknownBackendType             = errors.New("unknown backend type")
	ErrCreateBackendOptionsRequired   = errors.New("create backend options required")
	ErrUpdateBackendOptionsRequired   = errors.New("update backend options required")
	ErrUpdateBackendFieldMaskRequired = errors.New("update backend field mask required")
	ErrBackendNameRequired            = errors.New("name is required")
	ErrBackendDescriptionTooLong      = errors.New("description must not be longer than 255 characters")
	ErrBackendURLRequired             = errors.New("url is required")
)
