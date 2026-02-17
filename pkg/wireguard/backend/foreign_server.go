package backend

type ForeignServer struct {
	BackendId    string
	Interface    *ForeignInterface
	Name         string
	Description  string
	Type         string
	PublicKey    string
	ListenPort   int
	FirewallMark int
	Peers        []*Peer
}
