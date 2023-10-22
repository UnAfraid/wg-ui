package backend

import (
	"errors"
	"fmt"
)

type WireguardOptions struct {
	PrivateKey   string
	ListenPort   *int
	FirewallMark *int
	Peers        []*PeerOptions
}

func (o WireguardOptions) Validate() error {
	if len(o.PrivateKey) == 0 {
		return errors.New("private key required")
	}
	for _, peer := range o.Peers {
		if err := peer.Validate(); err != nil {
			return fmt.Errorf("peer: %w", err)
		}
	}
	return nil
}
