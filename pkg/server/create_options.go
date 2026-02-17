package server

type CreateOptions struct {
	Name         string
	Description  string
	BackendId    string
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
}
