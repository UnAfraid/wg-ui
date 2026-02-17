package driver

import "errors"

type InterfaceOptions struct {
	Name        string
	Description string
	Address     string
	Mtu         int
}

func (o InterfaceOptions) Validate() error {
	if len(o.Name) == 0 {
		return errors.New("name is required")
	}
	if len(o.Address) == 0 {
		return errors.New("address is required")
	}
	return nil
}
