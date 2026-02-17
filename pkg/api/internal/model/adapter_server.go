package model

import (
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/server"
)

func CreateServerInputToCreateServerOptions(input CreateServerInput) (*server.CreateOptions, error) {
	backendId, err := input.BackendID.String(IdKindBackend)
	if err != nil {
		return nil, err
	}

	return &server.CreateOptions{
		Name:         input.Name,
		Description:  adapt.Dereference(input.Description.Value()),
		BackendId:    backendId,
		Enabled:      adapt.Dereference(input.Enabled.Value()),
		PrivateKey:   adapt.Dereference(input.PrivateKey.Value()),
		ListenPort:   input.ListenPort.Value(),
		FirewallMark: input.FirewallMark.Value(),
		Address:      input.Address,
		DNS:          input.DNS.Value(),
		MTU:          adapt.Dereference(input.Mtu.Value()),
		Hooks:        adapt.Array(input.Hooks.Value(), ServerHookInputToServerHook),
	}, nil
}

func ToServer(server *server.Server) *Server {
	if server == nil {
		return nil
	}

	var backendRef *Backend
	if server.BackendId != "" {
		backendRef = &Backend{
			ID: StringID(IdKindBackend, server.BackendId),
		}
	}

	return &Server{
		ID:             StringID(IdKindServer, server.Id),
		Name:           server.Name,
		Description:    server.Description,
		Backend:        backendRef,
		Enabled:        server.Enabled,
		Running:        server.Running,
		PublicKey:      server.PublicKey,
		ListenPort:     server.ListenPort,
		FirewallMark:   server.FirewallMark,
		Address:        server.Address,
		DNS:            server.DNS,
		Mtu:            server.MTU,
		Hooks:          adapt.Array(server.Hooks, ToServerHook),
		InterfaceStats: ToServerInterfaceStats(server.Stats),
		CreateUser:     userIdToUser(server.CreateUserId),
		UpdateUser:     userIdToUser(server.UpdateUserId),
		DeleteUser:     userIdToUser(server.DeleteUserId),
		CreatedAt:      server.CreatedAt,
		UpdatedAt:      server.UpdatedAt,
		DeletedAt:      server.DeletedAt,
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

func UpdateServerInputToUpdateOptionsAndUpdateFieldMask(input UpdateServerInput) (options *server.UpdateOptions, fieldMask *server.UpdateFieldMask, err error) {
	fieldMask = &server.UpdateFieldMask{
		Description:  input.Description.IsSet(),
		BackendId:    input.BackendID.IsSet(),
		Enabled:      input.Enabled.IsSet(),
		PrivateKey:   input.PrivateKey.IsSet(),
		ListenPort:   input.ListenPort.IsSet(),
		FirewallMark: input.FirewallMark.IsSet(),
		Address:      input.Address.IsSet(),
		DNS:          input.DNS.IsSet(),
		MTU:          input.Mtu.IsSet(),
		Hooks:        input.Hooks.IsSet(),
	}

	var (
		description  string
		backendId    string
		enabled      bool
		privateKey   string
		listenPort   *int
		firewallMark *int
		address      string
		dns          []string
		mtu          int
		hooks        []*server.Hook
	)

	if fieldMask.Description {
		description = adapt.Dereference(input.Description.Value())
	}

	if fieldMask.BackendId {
		backendIdPtr := input.BackendID.Value()
		if backendIdPtr != nil {
			backendId, err = backendIdPtr.String(IdKindBackend)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	if fieldMask.Enabled {
		enabled = adapt.Dereference(input.Enabled.Value())
	}

	if fieldMask.PrivateKey {
		privateKey = adapt.Dereference(input.PrivateKey.Value())
	}

	if fieldMask.ListenPort {
		listenPort = input.ListenPort.Value()
	}

	if fieldMask.FirewallMark {
		firewallMark = input.FirewallMark.Value()
	}

	if fieldMask.Address {
		address = adapt.Dereference(input.Address.Value())
	}

	if fieldMask.DNS {
		dns = input.DNS.Value()
	}

	if fieldMask.MTU {
		mtu = adapt.Dereference(input.Mtu.Value())
	}

	if fieldMask.Hooks {
		hooks = adapt.Array(input.Hooks.Value(), ServerHookInputToServerHook)
	}

	options = &server.UpdateOptions{
		Description:  description,
		BackendId:    backendId,
		Enabled:      enabled,
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

func ToServerInterfaceStats(stats server.Stats) *ServerInterfaceStats {
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
