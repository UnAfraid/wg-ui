package driver

import "fmt"

type ConfigureOptions struct {
	InterfaceOptions InterfaceOptions
	WireguardOptions WireguardOptions
}

func (o ConfigureOptions) Validate() error {
	if err := o.InterfaceOptions.Validate(); err != nil {
		return fmt.Errorf("interface options: %w", err)
	}

	if err := o.WireguardOptions.Validate(); err != nil {
		return fmt.Errorf("wireguard options: %w", err)
	}

	return nil
}
