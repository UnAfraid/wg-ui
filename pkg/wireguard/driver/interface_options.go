package driver

import "errors"

type InterfaceOptions struct {
	Name        string
	Description string
	Address     string
	DNS         []string
	Mtu         int
	Hooks       []*HookOptions
}

func (o InterfaceOptions) Validate() error {
	if len(o.Name) == 0 {
		return errors.New("name is required")
	}
	if len(o.Address) == 0 {
		return errors.New("address is required")
	}
	for _, hook := range o.Hooks {
		if err := hook.Validate(); err != nil {
			return err
		}
	}
	return nil
}
