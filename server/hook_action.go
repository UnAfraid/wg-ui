package server

type HookAction string

var (
	HookActionCreate HookAction = "CREATE"
	HookActionUpdate HookAction = "UPDATE"
	HookActionDelete HookAction = "DELETE"
	HookActionStart  HookAction = "START"
	HookActionStop   HookAction = "STOP"
)
