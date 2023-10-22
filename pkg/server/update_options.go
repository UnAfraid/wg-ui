package server

type UpdateOptions struct {
	Description  string
	Enabled      bool
	Running      bool
	PrivateKey   string
	ListenPort   *int
	FirewallMark *int
	Address      string
	DNS          []string
	MTU          int
	Stats        Stats
	Hooks        []*Hook
	CreateUserId string
	UpdateUserId string
}
