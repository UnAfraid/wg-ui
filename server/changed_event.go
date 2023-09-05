package server

const (
	ChangedActionCreated = "CREATED"
	ChangedActionUpdated = "UPDATED"
	ChangedActionDeleted = "DELETED"
)

type ChangedEvent struct {
	Action string
	Server *Server
}
