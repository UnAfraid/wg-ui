package peer

type CreateOptions struct {
	Name                string
	Description         string
	PublicKey           string
	Endpoint            string
	AllowedIPs          []string
	PresharedKey        string
	PersistentKeepalive int
	Hooks               []*Hook
}
