package backend

type ForeignServer struct {
	Interface    *ForeignInterface
	Name         string
	Type         string
	PublicKey    string
	ListenPort   int
	FirewallMark int
	Peers        []*Peer
}
