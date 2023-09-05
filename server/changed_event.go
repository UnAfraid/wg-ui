package server

const (
	ChangedActionCreated               = "CREATED"
	ChangedActionUpdated               = "UPDATED"
	ChangedActionDeleted               = "DELETED"
	ChangedActionInterfaceStatsUpdated = "INTERFACE_STATS_UPDATED"
	ChangedActionStarted               = "STARTED"
	ChangedActionStopped               = "STOPPED"
)

type ChangedEvent struct {
	Action string
	Server *Server
}
