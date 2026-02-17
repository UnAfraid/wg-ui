//go:build !linux && !darwin

package exec

import (
	"context"
	"errors"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func Register() {
	driver.Register("exec", func(_ context.Context, rawURL string) (driver.Backend, error) {
		return NewExecBackend(rawURL)
	}, false)
}

func NewExecBackend(_ string) (driver.Backend, error) {
	return nil, errors.New("exec backend is only supported on linux and darwin")
}
