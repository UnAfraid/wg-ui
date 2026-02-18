package driver

type ForeignServer struct {
	BackendId    string
	Interface    *ForeignInterface
	Hooks        []*HookOptions
	Name         string
	Description  string
	Type         string
	PublicKey    string
	ListenPort   int
	FirewallMark int
	Peers        []*Peer
}
