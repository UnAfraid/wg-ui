package server

type NameOption struct {
	Name string
}

func (option *NameOption) Validate() error {
	if len(option.Name) == 0 {
		return ErrNameRequired
	}
	return nil
}
