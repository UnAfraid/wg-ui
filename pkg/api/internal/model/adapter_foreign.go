package model

import (
	"net"

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func ToForeignInterface(foreignInterface *driver.ForeignInterface) *ForeignInterface {
	if foreignInterface == nil {
		return nil
	}

	return &ForeignInterface{
		Name:      foreignInterface.Name,
		Addresses: foreignInterface.Addresses,
		Mtu:       foreignInterface.Mtu,
	}
}

func ToForeignServer(foreignServer *driver.ForeignServer) *ForeignServer {
	if foreignServer == nil {
		return nil
	}

	return &ForeignServer{
		ForeignInterface: ToForeignInterface(foreignServer.Interface),
		Name:             foreignServer.Name,
		Type:             foreignServer.Type,
		PublicKey:        foreignServer.PublicKey,
		ListenPort:       foreignServer.ListenPort,
		FirewallMark:     foreignServer.FirewallMark,
		Peers:            adapt.Array(foreignServer.Peers, ToForeignPeer),
		Backend: &Backend{
			ID: StringID(IdKindBackend, foreignServer.BackendId),
		},
	}
}

func ToForeignPeer(foreignPeer *driver.Peer) *ForeignPeer {
	if foreignPeer == nil {
		return nil
	}

	return &ForeignPeer{
		PublicKey: foreignPeer.PublicKey,
		Endpoint:  adapt.ToPointerNilZero(foreignPeer.Endpoint),
		AllowedIps: adapt.Array(foreignPeer.AllowedIPs, func(allowedIp net.IPNet) string {
			return allowedIp.String()
		}),
		PersistentKeepAliveInterval: int(foreignPeer.PersistentKeepalive.Seconds()),
		LastHandshakeTime:           adapt.ToPointer(foreignPeer.Stats.LastHandshakeTime),
		ReceiveBytes:                float64(foreignPeer.Stats.ReceiveBytes),
		TransmitBytes:               float64(foreignPeer.Stats.TransmitBytes),
		ProtocolVersion:             foreignPeer.Stats.ProtocolVersion,
	}
}
