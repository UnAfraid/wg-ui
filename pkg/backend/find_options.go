package backend

type FindOptions struct {
	Ids          []string
	Type         *string
	Enabled      *bool
	Query        string
	CreateUserId *string
	UpdateUserId *string
}
