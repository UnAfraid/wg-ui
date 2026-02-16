//go:build !linux

package networkmanager

import (
	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

func init() {
	backend.Register("networkmanager", nil, false)
}
