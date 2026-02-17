package backend

type FindOneOptions struct {
	IdOption   *IdOption
	NameOption *NameOption
}

func (options *FindOneOptions) Validate() error {
	var optionsCount int
	if options.IdOption != nil {
		optionsCount++
		if err := options.IdOption.Validate(); err != nil {
			return err
		}
	}
	if options.NameOption != nil {
		optionsCount++
		if err := options.NameOption.Validate(); err != nil {
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
