package user

const (
	ChangedActionCreated = "CREATED"
	ChangedActionUpdated = "UPDATED"
	ChangedActionDeleted = "DELETED"
)

type ChangedEvent struct {
	Action string
	User   *User
}
