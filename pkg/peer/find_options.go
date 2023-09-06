package peer

type FindOptions struct {
	Ids          []string
	ServerId     *string
	CreateUserId *string
	UpdateUserId *string
	Query        string
}
