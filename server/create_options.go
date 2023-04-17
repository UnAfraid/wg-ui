package server

type CreateOptions struct {
	Name         string
	Description  string
	Enabled      bool
	PublicKey    string
	PrivateKey   string
	ListenPort   *int
	FirewallMark *int
	Address      string
	DNS          []string
	MTU          int
	Hooks        []*Hook
}
