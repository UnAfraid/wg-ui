package user

type FindOneOptions struct {
	IdOption    *IdOption
	EmailOption *EmailOption
}

func (options *FindOneOptions) Validate() error {
	var optionsCount int
	if options.IdOption != nil {
		optionsCount++
		if err := options.IdOption.Validate(); err != nil {
			return err
		}
	}

	if options.EmailOption != nil {
		optionsCount++
		if err := options.EmailOption.Validate(); err != nil {
			return err
		}
	}

	if optionsCount == 0 {
		return ErrOneOptionRequired
	} else if optionsCount != 1 {
		return ErrOnlyOneOptionAllowed
	}

	return nil
}
