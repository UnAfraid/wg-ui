//go:build !linux

package linux

import (
	"errors"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

func NewLinuxBackend() (backend.Backend, error) {
	return nil, errors.New("linux backend is only supported on linux")
}
