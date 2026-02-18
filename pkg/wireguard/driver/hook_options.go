package driver

import "errors"

type HookOptions struct {
	Command       string
	RunOnPreUp    bool
	RunOnPostUp   bool
	RunOnPreDown  bool
	RunOnPostDown bool
}

func (o *HookOptions) Validate() error {
	if o == nil {
		return errors.New("hook options are required")
	}
	if o.Command == "" {
		return errors.New("hook command is required")
	}
	if !(o.RunOnPreUp || o.RunOnPostUp || o.RunOnPreDown || o.RunOnPostDown) {
		return errors.New("hook must be enabled for at least one lifecycle event")
	}
	return nil
}
