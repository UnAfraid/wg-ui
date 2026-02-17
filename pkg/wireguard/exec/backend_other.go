//go:build !linux && !darwin

package exec

import "github.com/UnAfraid/wg-ui/pkg/wireguard/backend"

func init() {
	backend.Register("exec", nil, false)
}
