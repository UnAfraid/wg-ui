package server

type UpdateFieldMask struct {
	Description  bool
	BackendId    bool
	Enabled      bool
	Running      bool
	PrivateKey   bool
	ListenPort   bool
	FirewallMark bool
	Address      bool
	DNS          bool
	MTU          bool
	Stats        bool
	Hooks        bool
	CreateUserId bool
	UpdateUserId bool
}
