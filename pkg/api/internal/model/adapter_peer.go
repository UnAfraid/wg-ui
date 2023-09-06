package model

import (
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"github.com/UnAfraid/wg-ui/pkg/wg"
)

func CreatePeerInputToCreateOptions(input CreatePeerInput) *peer.CreateOptions {
	return &peer.CreateOptions{
		Name:                input.Name,
		Description:         adapt.Dereference(input.Description.Value()),
		PublicKey:           input.PublicKey,
		Endpoint:            adapt.Dereference(input.Endpoint.Value()),
		AllowedIPs:          input.AllowedIPs,
		PresharedKey:        adapt.Dereference(input.PresharedKey.Value()),
		PersistentKeepalive: adapt.Dereference(input.PersistentKeepalive.Value()),
		Hooks:               adapt.Array(input.Hooks.Value(), PeerHookInputToPeerHook),
	}
}

func UpdatePeerInputToUpdatePeerOptionsAndUpdatePeerFieldMask(input UpdatePeerInput) (options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask) {
	fieldMask = &peer.UpdateFieldMask{
		Name:                input.Name.IsSet(),
		Description:         input.Description.IsSet(),
		PublicKey:           input.PublicKey.IsSet(),
		Endpoint:            input.Endpoint.IsSet(),
		AllowedIPs:          input.AllowedIPs.IsSet(),
		PresharedKey:        input.PresharedKey.IsSet(),
		PersistentKeepalive: input.PersistentKeepalive.IsSet(),
		Hooks:               input.Hooks.IsSet(),
	}

	var (
		name                string
		description         string
		enabled             bool
		publicKey           string
		allowedIPs          []string
		endpoint            string
		presharedKey        string
		persistentKeepalive int
		hooks               []*peer.Hook
	)

	if fieldMask.Name {
		name = adapt.Dereference(input.Name.Value())
	}

	if fieldMask.Description {
		description = adapt.Dereference(input.Description.Value())
	}

	if fieldMask.PublicKey {
		publicKey = adapt.Dereference(input.PublicKey.Value())
	}

	if fieldMask.Endpoint {
		endpoint = adapt.Dereference(input.Endpoint.Value())
	}

	if fieldMask.AllowedIPs {
		allowedIPs = input.AllowedIPs.Value()
	}

	if fieldMask.PresharedKey {
		presharedKey = adapt.Dereference(input.PresharedKey.Value())
	}

	if fieldMask.PersistentKeepalive {
		persistentKeepalive = adapt.Dereference(input.PersistentKeepalive.Value())
	}

	if fieldMask.Hooks {
		hooks = adapt.Array(input.Hooks.Value(), PeerHookInputToPeerHook)
	}

	options = &peer.UpdateOptions{
		Name:                name,
		Description:         description,
		Enabled:             enabled,
		PublicKey:           publicKey,
		Endpoint:            endpoint,
		AllowedIPs:          allowedIPs,
		PresharedKey:        presharedKey,
		PersistentKeepalive: persistentKeepalive,
		Hooks:               hooks,
	}

	return options, fieldMask
}

func ToPeer(peer *peer.Peer) *Peer {
	if peer == nil {
		return nil
	}
	return &Peer{
		ID: StringID(IdKindPeer, peer.Id),
		Server: &Server{
			ID: StringID(IdKindServer, peer.ServerId),
		},
		Name:                peer.Name,
		Description:         peer.Description,
		PublicKey:           peer.PublicKey,
		Endpoint:            peer.Endpoint,
		AllowedIPs:          peer.AllowedIPs,
		PresharedKey:        peer.PresharedKey,
		PersistentKeepalive: adapt.ToPointerNilZero(peer.PersistentKeepalive),
		Hooks:               adapt.Array(peer.Hooks, ToPeerHook),
		CreateUser:          userIdToUser(peer.CreateUserId),
		UpdateUser:          userIdToUser(peer.UpdateUserId),
		DeleteUser:          userIdToUser(peer.DeleteUserId),
		CreatedAt:           peer.CreatedAt,
		UpdatedAt:           peer.UpdatedAt,
		DeletedAt:           peer.DeletedAt,
	}
}

func ToPeerHook(hook *peer.Hook) *PeerHook {
	if hook == nil {
		return nil
	}
	return &PeerHook{
		Command:     hook.Command,
		RunOnCreate: hook.RunOnCreate,
		RunOnUpdate: hook.RunOnUpdate,
		RunOnDelete: hook.RunOnDelete,
	}
}

func PeerHookInputToPeerHook(hook *PeerHookInput) *peer.Hook {
	if hook == nil {
		return nil
	}
	return &peer.Hook{
		Command:     hook.Command,
		RunOnCreate: hook.RunOnCreate,
		RunOnUpdate: hook.RunOnUpdate,
		RunOnDelete: hook.RunOnDelete,
	}
}

func ToPeerStats(stats *wg.PeerStats) *PeerStats {
	if stats == nil {
		return nil
	}
	return &PeerStats{
		LastHandshakeTime: adapt.ToPointer(stats.LastHandshakeTime),
		ReceiveBytes:      float64(stats.ReceiveBytes),
		TransmitBytes:     float64(stats.TransmitBytes),
		ProtocolVersion:   stats.ProtocolVersion,
	}
}
