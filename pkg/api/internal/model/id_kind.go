package model

type IdKind string

const (
	IdKindUser    IdKind = "User"
	IdKindServer  IdKind = "Server"
	IdKindPeer    IdKind = "Peer"
	IdKindBackend IdKind = "Backend"
)

func (ik IdKind) String() string {
	return string(ik)
}
