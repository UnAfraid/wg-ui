package wg

type ForeignServer struct {
	ForeignInterface *ForeignInterface
	Name             string
	Type             string
	PublicKey        string
	ListenPort       int
	FirewallMark     int
	Peers            []*ForeignPeer
}
