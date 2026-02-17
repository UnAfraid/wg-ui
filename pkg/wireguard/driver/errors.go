package driver

import "errors"

var (
	// ErrConnectionStale signals that backend runtime connection state is no longer usable
	// and the backend instance should be recreated and retried once.
	ErrConnectionStale = errors.New("wireguard backend connection is stale")
)
