package backend

type Wireguard struct {
	Name         string
	PublicKey    string
	PrivateKey   string
	ListenPort   int
	FirewallMark int
	Peers        []*Peer
}
