package peer

import (
	"errors"
)

var (
	ErrIdRequired                  = errors.New("id is required")
	ErrServerIdRequired            = errors.New("server id is required")
	ErrNameRequired                = errors.New("name is required")
	ErrOneOptionRequired           = errors.New("one option is required")
	ErrOnlyOneOptionAllowed        = errors.New("only one option is allowed")
	ErrServerNotFound              = errors.New("server not found")
	ErrPeerNotFound                = errors.New("peer not found")
	ErrPeerIdAlreadyExists         = errors.New("peer id already exists")
	ErrPeerNameAlreadyInUse        = errors.New("peer name already in use")
	ErrPublicKeyRequired           = errors.New("public key is required")
	ErrPublicKeyAlreadyExists      = errors.New("public key already exists")
	ErrCreatePeerOptionsRequired   = errors.New("create peer options are required")
	ErrUpdatePeerOptionsRequired   = errors.New("update peer options are required")
	ErrUpdatePeerFieldMaskRequired = errors.New("update peer field mask are required")
)
