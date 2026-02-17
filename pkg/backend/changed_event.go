package backend

const (
	ChangedActionCreated  = "CREATED"
	ChangedActionUpdated  = "UPDATED"
	ChangedActionDeleted  = "DELETED"
	ChangedActionEnabled  = "ENABLED"
	ChangedActionDisabled = "DISABLED"
)

type ChangedEvent struct {
	Action  string   `json:"action"`
	Backend *Backend `json:"backend"`
}
