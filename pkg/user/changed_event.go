package user

const (
	ChangedActionCreated = "CREATED"
	ChangedActionUpdated = "UPDATED"
	ChangedActionDeleted = "DELETED"
)

type ChangedEvent struct {
	Action string `json:"action"`
	User   *User  `json:"user"`
}
