package builtin

import (
	"github.com/UnAfraid/wg-ui/pkg/wireguard/exec"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/linux"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/networkmanager"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/routeros"
)

// RegisterAll registers all built-in wireguard backend implementations.
func RegisterAll() {
	exec.Register()
	linux.Register()
	networkmanager.Register()
	routeros.Register()
}
