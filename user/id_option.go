package user

type IdOption struct {
	Id          string
	WithDeleted bool
}

func (option *IdOption) Validate() error {
	if len(option.Id) == 0 {
		return ErrIdRequired
	}
	return nil
}
