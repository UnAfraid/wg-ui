package driver

import (
	"net"
	"time"
)

type Peer struct {
	Name                string
	Description         string
	PublicKey           string
	Endpoint            string
	AllowedIPs          []net.IPNet
	PresharedKey        string
	PersistentKeepalive time.Duration
	Stats               PeerStats
}
