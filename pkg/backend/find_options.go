package backend

import "errors"

type FindOneOptions struct {
	IdOption   *IdOption
	NameOption *NameOption
}

func (o *FindOneOptions) Validate() error {
	if o.IdOption == nil && o.NameOption == nil {
		return errors.New("at least one option is required")
	}
	return nil
}

type IdOption struct {
	Id string
}

type NameOption struct {
	Name string
}

type FindOptions struct {
	Ids          []string
	Type         *string
	Enabled      *bool
	Query        string
	CreateUserId *string
	UpdateUserId *string
}
