package peer

type UpdateFieldMask struct {
	Name                bool
	Description         bool
	PublicKey           bool
	Endpoint            bool
	AllowedIPs          bool
	PresharedKey        bool
	PersistentKeepalive bool
	Hooks               bool
	UpdateUserId        bool
}
