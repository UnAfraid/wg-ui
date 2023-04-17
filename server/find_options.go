package server

type FindOptions struct {
	Ids          []string
	Query        string
	Enabled      *bool
	CreateUserId *string
	UpdateUserId *string
}
