package driver

import "errors"

type PeerOptions struct {
	PublicKey           string
	Endpoint            string
	AllowedIPs          []string
	PresharedKey        string
	PersistentKeepalive int
}

func (o *PeerOptions) Validate() error {
	if len(o.PublicKey) == 0 {
		return errors.New("public key is required")
	}

	if len(o.AllowedIPs) == 0 {
		return errors.New("allowed ips are required")
	}

	return nil
}
