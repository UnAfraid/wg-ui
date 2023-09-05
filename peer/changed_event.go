package peer

const (
	ChangedActionCreated = "CREATED"
	ChangedActionUpdated = "UPDATED"
	ChangedActionDeleted = "DELETED"
)

type ChangedEvent struct {
	Action string
	Peer   *Peer
}
