//go:build !linux

package wg

import (
	"github.com/UnAfraid/wg-ui/pkg/server"
)

func configureInterface(name string, address string, mtu int) error {
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
