package model

import (
	"context"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/UnAfraid/wg-ui/wg"
)

func CreatePeerInputToCreateOptions(input CreatePeerInput) *peer.CreateOptions {
	return &peer.CreateOptions{
		Name:                input.Name,
		Description:         adapt.Dereference(input.Description),
		PublicKey:           input.PublicKey,
		Endpoint:            adapt.Dereference(input.Endpoint),
		AllowedIPs:          input.AllowedIPs,
		PresharedKey:        adapt.Dereference(input.PresharedKey),
		PersistentKeepalive: adapt.Dereference(input.PersistentKeepalive),
		Hooks:               adapt.Array(input.Hooks, PeerHookInputToPeerHook),
	}
}

func UpdatePeerInputToUpdatePeerOptionsAndUpdatePeerFieldMask(ctx context.Context, input UpdatePeerInput) (options *peer.UpdateOptions, fieldMask *peer.UpdateFieldMask) {
	fieldMask = &peer.UpdateFieldMask{
		Name:                resolverHasArgumentField(ctx, "input", "name"),
		Description:         resolverHasArgumentField(ctx, "input", "description"),
		PublicKey:           resolverHasArgumentField(ctx, "input", "publicKey"),
		Endpoint:            resolverHasArgumentField(ctx, "input", "endpoint"),
		AllowedIPs:          resolverHasArgumentField(ctx, "input", "allowedIPs"),
		PresharedKey:        resolverHasArgumentField(ctx, "input", "presharedKey"),
		PersistentKeepalive: resolverHasArgumentField(ctx, "input", "persistentKeepalive"),
		Hooks:               resolverHasArgumentField(ctx, "input", "hooks"),
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
		name = adapt.Dereference(input.Name)
	}

	if fieldMask.Description {
		description = adapt.Dereference(input.Description)
	}

	if fieldMask.PublicKey {
		publicKey = adapt.Dereference(input.PublicKey)
	}

	if fieldMask.Endpoint {
		endpoint = adapt.Dereference(input.Endpoint)
	}

	if fieldMask.AllowedIPs {
		allowedIPs = input.AllowedIPs
	}

	if fieldMask.PresharedKey {
		presharedKey = adapt.Dereference(input.PresharedKey)
	}

	if fieldMask.PersistentKeepalive {
		persistentKeepalive = adapt.Dereference(input.PersistentKeepalive)
	}

	if fieldMask.Hooks {
		hooks = adapt.Array(input.Hooks, PeerHookInputToPeerHook)
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
