//go:build !darwin

package darwin

import (
	"errors"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

func init() {
	backend.Register("darwin", nil, false)
}

func NewDarwinBackend() (backend.Backend, error) {
	return nil, errors.New("darwin backend is only supported on macOS")
}
