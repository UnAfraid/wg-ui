//go:build !linux

package wg

func configureInterface(name string, address string, mtu int) error {
	return nil
}

func deleteInterface(name string) error {
	return nil
}

func interfaceStats(name string) (*InterfaceStats, error) {
	return nil, nil
}

func findForeignInterfaces(knownInterfaces []string) ([]ForeignInterface, error) {
	return nil, nil
}
