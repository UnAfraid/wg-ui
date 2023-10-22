package backend

import (
	"net"
	"time"
)

type Peer struct {
	PublicKey           string
	Endpoint            string
	AllowedIPs          []net.IPNet
	PresharedKey        string
	PersistentKeepalive time.Duration
	Stats               PeerStats
}
