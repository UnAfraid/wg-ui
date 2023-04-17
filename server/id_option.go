package server

type IdOption struct {
	Id string
}

func (option *IdOption) Validate() error {
	if len(option.Id) == 0 {
		return ErrIdRequired
	}
	return nil
}
