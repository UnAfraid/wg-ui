//go:build !linux

package wg

import (
	"net"

	"github.com/UnAfraid/wg-ui/pkg/server"
)

func configureInterface(name string, address string, mtu int) error {
	return nil
}

func configureRoutes(name string, allowedIPs []net.IPNet) error {
	return nil
}

func deleteInterface(name string) error {
	return nil
}

func interfaceStats(name string) (server.Stats, error) {
	return server.Stats{}, nil
}

func findForeignInterfaces(knownInterfaces []string) ([]ForeignInterface, error) {
	return nil, nil
}
