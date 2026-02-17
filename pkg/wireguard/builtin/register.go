package builtin

import (
	"github.com/UnAfraid/wg-ui/pkg/wireguard/exec"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/linux"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/networkmanager"
)

// RegisterAll registers all built-in wireguard backend implementations.
func RegisterAll() {
	exec.Register()
	linux.Register()
	networkmanager.Register()
}
