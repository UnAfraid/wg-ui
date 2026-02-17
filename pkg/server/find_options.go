package server

type FindOptions struct {
	Ids          []string
	Query        string
	BackendId    *string
	Enabled      *bool
	CreateUserId *string
	UpdateUserId *string
}
