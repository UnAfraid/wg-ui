package driver

import (
	"time"
)

type PeerStats struct {
	Endpoint          string
	LastHandshakeTime time.Time
	ReceiveBytes      int64
	TransmitBytes     int64
	ProtocolVersion   int
}
