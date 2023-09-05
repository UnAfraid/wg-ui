package server

type CreateOptions struct {
	Name         string
	Description  string
	Enabled      bool
	Running      bool
	PublicKey    string
	PrivateKey   string
	ListenPort   *int
	FirewallMark *int
	Address      string
	DNS          []string
	MTU          int
	Stats        Stats
	Hooks        []*Hook
}
