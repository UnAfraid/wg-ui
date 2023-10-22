package backend

import (
	"time"
)

type PeerStats struct {
	LastHandshakeTime time.Time
	ReceiveBytes      int64
	TransmitBytes     int64
	ProtocolVersion   int
}
