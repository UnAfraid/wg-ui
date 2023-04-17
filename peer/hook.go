package peer

type Hook struct {
	Command     string
	RunOnCreate bool
	RunOnUpdate bool
	RunOnDelete bool
}

func (h *Hook) ShouldExecute(action HookAction) bool {
	switch action {
	case HookActionCreate:
		return h.RunOnCreate
	case HookActionUpdate:
		return h.RunOnUpdate
	case HookActionDelete:
		return h.RunOnDelete
	}
	return false
}
