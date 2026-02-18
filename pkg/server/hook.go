package server

type Hook struct {
	Command       string
	RunOnPreUp    bool
	RunOnPostUp   bool
	RunOnPreDown  bool
	RunOnPostDown bool

	// Legacy fields kept for backward compatibility with existing stored data.
	RunOnCreate bool
	RunOnUpdate bool
	RunOnDelete bool
	RunOnStart  bool
	RunOnStop   bool
}

func (h *Hook) shouldExecute(action HookAction) bool {
	switch action {
	case HookActionPreUp:
		return h.RunOnPreUp
	case HookActionPostUp:
		return h.RunOnPostUp || h.RunOnStart
	case HookActionPreDown:
		return h.RunOnPreDown
	case HookActionPostDown:
		return h.RunOnPostDown || h.RunOnStop
	}
	return false
}
