package server

type HookAction string

var (
	HookActionPreUp    HookAction = "PRE_UP"
	HookActionPostUp   HookAction = "POST_UP"
	HookActionPreDown  HookAction = "PRE_DOWN"
	HookActionPostDown HookAction = "POST_DOWN"
)
