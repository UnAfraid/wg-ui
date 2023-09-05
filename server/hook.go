package server

type Hook struct {
	Command     string
	RunOnCreate bool
	RunOnUpdate bool
	RunOnDelete bool
	RunOnStart  bool
	RunOnStop   bool
}

func (h *Hook) shouldExecute(action HookAction) bool {
	switch action {
	case HookActionCreate:
		return h.RunOnCreate
	case HookActionUpdate:
		return h.RunOnUpdate
	case HookActionDelete:
		return h.RunOnDelete
	case HookActionStart:
		return h.RunOnStart
	case HookActionStop:
		return h.RunOnStop
	}
	return false
}
