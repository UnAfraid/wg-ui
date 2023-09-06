package user

type EmailOption struct {
	Email string
}

func (option *EmailOption) Validate() error {
	if len(option.Email) == 0 {
		return ErrEmailRequired
	}
	return nil
}
