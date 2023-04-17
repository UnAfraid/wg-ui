package peer

type UpdateOptions struct {
	Name                string
	Description         string
	Enabled             bool
	PublicKey           string
	Endpoint            string
	AllowedIPs          []string
	PresharedKey        string
	PersistentKeepalive int
	Hooks               []*Hook
}
