package peer

type HookAction string

var (
	HookActionCreate HookAction = "CREATE"
	HookActionUpdate HookAction = "UPDATE"
	HookActionDelete HookAction = "DELETE"
)
