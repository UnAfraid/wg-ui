package server

type UpdateOptions struct {
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
	Hooks        []*Hook
}
