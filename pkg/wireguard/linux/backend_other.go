//go:build !linux

package linux

import (
	"context"
	"errors"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func Register() {
	driver.Register("linux", func(_ context.Context, rawURL string) (driver.Backend, error) {
		return NewLinuxBackend(rawURL)
	}, false)
}

func NewLinuxBackend(_ string) (driver.Backend, error) {
	return nil, errors.New("linux backend is only supported on linux")
}
