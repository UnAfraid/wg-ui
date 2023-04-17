package wg

import (
	"time"
)

type ForeignPeer struct {
	PublicKey                   string
	Endpoint                    *string
	AllowedIPs                  []string
	PersistentKeepaliveInterval float64
	LastHandshakeTime           time.Time
	ReceiveBytes                int64
	TransmitBytes               int64
	ProtocolVersion             int
}
