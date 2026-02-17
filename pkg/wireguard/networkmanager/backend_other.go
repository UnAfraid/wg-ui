//go:build !linux

package networkmanager

import (
	"context"
	"errors"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func Register() {
	driver.Register("networkmanager", func(_ context.Context, rawURL string) (driver.Backend, error) {
		return NewNetworkManagerBackend(rawURL)
	}, false)
}

func NewNetworkManagerBackend(_ string) (driver.Backend, error) {
	return nil, errors.New("networkmanager backend is only supported on linux")
}
