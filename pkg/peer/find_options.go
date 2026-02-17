package peer

type FindOptions struct {
	Ids          []string
	ServerId     *string
	ServerIds    []string
	CreateUserId *string
	UpdateUserId *string
	Query        string
}
