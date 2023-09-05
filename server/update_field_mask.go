package server

type UpdateFieldMask struct {
	Description  bool
	Enabled      bool
	Running      bool
	PublicKey    bool
	PrivateKey   bool
	ListenPort   bool
	FirewallMark bool
	Address      bool
	DNS          bool
	MTU          bool
	Stats        bool
	Hooks        bool
	UpdateUserId bool
}
