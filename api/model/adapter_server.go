package model

import (
	"context"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/UnAfraid/wg-ui/wg"
)

func CreateServerInputToCreateServerOptions(input CreateServerInput) (_ *server.CreateOptions, err error) {
	return &server.CreateOptions{
		Name:         input.Name,
		Description:  adapt.Dereference(input.Description),
		Enabled:      adapt.Dereference(input.Enabled),
		PublicKey:    adapt.Dereference(input.PublicKey),
		PrivateKey:   adapt.Dereference(input.PrivateKey),
		ListenPort:   input.ListenPort,
		FirewallMark: input.FirewallMark,
		Address:      input.Address,
		DNS:          input.DNS,
		MTU:          adapt.Dereference(input.Mtu),
		Hooks:        adapt.Array(input.Hooks, ServerHookInputToServerHook),
	}, nil
}

func ToServer(server *server.Server) *Server {
	if server == nil {
		return nil
	}

	return &Server{
		ID:           StringID(IdKindServer, server.Id),
		Name:         server.Name,
		Description:  server.Description,
		Enabled:      server.Enabled,
		Running:      server.Running,
		PublicKey:    server.PublicKey,
		ListenPort:   server.ListenPort,
		FirewallMark: server.FirewallMark,
		Address:      server.Address,
		DNS:          server.DNS,
		Mtu:          server.MTU,
		Hooks:        adapt.Array(server.Hooks, ToServerHook),
		CreateUser:   userIdToUser(server.CreateUserId),
		UpdateUser:   userIdToUser(server.UpdateUserId),
		DeleteUser:   userIdToUser(server.DeleteUserId),
		CreatedAt:    server.CreatedAt,
		UpdatedAt:    server.UpdatedAt,
		DeletedAt:    server.DeletedAt,
	}
}

func ToServerHook(hook *server.Hook) *ServerHook {
	if hook == nil {
		return nil
	}
	return &ServerHook{
		Command:     hook.Command,
		RunOnCreate: hook.RunOnCreate,
		RunOnUpdate: hook.RunOnUpdate,
		RunOnDelete: hook.RunOnDelete,
		RunOnStart:  hook.RunOnStart,
		RunOnStop:   hook.RunOnStop,
	}
}

func ServerHookInputToServerHook(hook *ServerHookInput) *server.Hook {
	if hook == nil {
		return nil
	}
	return &server.Hook{
		Command:     hook.Command,
		RunOnCreate: hook.RunOnCreate,
		RunOnUpdate: hook.RunOnUpdate,
		RunOnDelete: hook.RunOnDelete,
		RunOnStart:  hook.RunOnStart,
		RunOnStop:   hook.RunOnStop,
	}
}

func UpdateServerInputToUpdateOptionsAndUpdateFieldMask(ctx context.Context, input UpdateServerInput) (options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, err error) {
	fieldMask = &server.UpdateFieldMask{
		Description:  resolverHasArgumentField(ctx, "input", "description"),
		Enabled:      resolverHasArgumentField(ctx, "input", "enabled"),
		PublicKey:    resolverHasArgumentField(ctx, "input", "publicKey"),
		PrivateKey:   resolverHasArgumentField(ctx, "input", "privateKey"),
		ListenPort:   resolverHasArgumentField(ctx, "input", "listenPort"),
		FirewallMark: resolverHasArgumentField(ctx, "input", "firewallMark"),
		Address:      resolverHasArgumentField(ctx, "input", "address"),
		DNS:          resolverHasArgumentField(ctx, "input", "dns"),
		MTU:          resolverHasArgumentField(ctx, "input", "mtu"),
		Hooks:        resolverHasArgumentField(ctx, "input", "hooks"),
	}

	var (
		description  string
		enabled      bool
		publicKey    string
		privateKey   string
		listenPort   *int
		firewallMark *int
		address      string
		dns          []string
		mtu          int
		hooks        []*server.Hook
	)

	if fieldMask.Description {
		description = adapt.Dereference(input.Description)
	}

	if fieldMask.Enabled {
		enabled = adapt.Dereference(input.Enabled)
	}

	if fieldMask.PublicKey {
		publicKey = adapt.Dereference(input.PublicKey)
	}

	if fieldMask.PrivateKey {
		privateKey = adapt.Dereference(input.PrivateKey)
	}

	if fieldMask.ListenPort {
		listenPort = input.ListenPort
	}

	if fieldMask.FirewallMark {
		firewallMark = input.FirewallMark
	}

	if fieldMask.Address {
		address = adapt.Dereference(input.Address)
	}

	if fieldMask.DNS {
		dns = input.DNS
	}

	if fieldMask.MTU {
		mtu = adapt.Dereference(input.Mtu)
	}

	if fieldMask.Hooks {
		hooks = adapt.Array(input.Hooks, ServerHookInputToServerHook)
	}

	options = &server.UpdateOptions{
		Description:  description,
		Enabled:      enabled,
		PublicKey:    publicKey,
		PrivateKey:   privateKey,
		ListenPort:   listenPort,
		FirewallMark: firewallMark,
		Address:      address,
		DNS:          dns,
		MTU:          mtu,
		Hooks:        hooks,
	}

	return options, fieldMask, nil
}

func ToServerInterfaceStats(stats *wg.InterfaceStats) *ServerInterfaceStats {
	if stats == nil {
		return nil
	}
	return &ServerInterfaceStats{
		RxPackets:         float64(stats.RxPackets),
		TxPackets:         float64(stats.TxPackets),
		RxBytes:           float64(stats.RxBytes),
		TxBytes:           float64(stats.TxBytes),
		RxErrors:          float64(stats.RxErrors),
		TxErrors:          float64(stats.TxErrors),
		RxDropped:         float64(stats.RxDropped),
		TxDropped:         float64(stats.TxDropped),
		Multicast:         float64(stats.Multicast),
		Collisions:        float64(stats.Collisions),
		RxLengthErrors:    float64(stats.RxLengthErrors),
		RxOverErrors:      float64(stats.RxOverErrors),
		RxCrcErrors:       float64(stats.RxCrcErrors),
		RxFrameErrors:     float64(stats.RxFrameErrors),
		RxFifoErrors:      float64(stats.RxFifoErrors),
		RxMissedErrors:    float64(stats.RxMissedErrors),
		TxAbortedErrors:   float64(stats.TxAbortedErrors),
		TxCarrierErrors:   float64(stats.TxCarrierErrors),
		TxFifoErrors:      float64(stats.TxFifoErrors),
		TxHeartbeatErrors: float64(stats.TxHeartbeatErrors),
		TxWindowErrors:    float64(stats.TxWindowErrors),
		RxCompressed:      float64(stats.RxCompressed),
		TxCompressed:      float64(stats.TxCompressed),
	}
}
