package model

import (
	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/wg"
)

func ToForeignInterface(foreignInterface *wg.ForeignInterface) *ForeignInterface {
	if foreignInterface == nil {
		return nil
	}

	return &ForeignInterface{
		Name:      foreignInterface.Name,
		Addresses: foreignInterface.Addresses,
		Mtu:       foreignInterface.Mtu,
	}
}

func ToForeignServer(foreignServer *wg.ForeignServer) *ForeignServer {
	if foreignServer == nil {
		return nil
	}

	return &ForeignServer{
		ForeignInterface: ToForeignInterface(foreignServer.ForeignInterface),
		Name:             foreignServer.Name,
		Type:             foreignServer.Type,
		PublicKey:        foreignServer.PublicKey,
		ListenPort:       foreignServer.ListenPort,
		FirewallMark:     foreignServer.FirewallMark,
		Peers:            adapt.Array(foreignServer.Peers, ToForeignPeer),
	}
}

func ToForeignPeer(foreignPeer *wg.ForeignPeer) *ForeignPeer {
	if foreignPeer == nil {
		return nil
	}

	return &ForeignPeer{
		PublicKey:                   foreignPeer.PublicKey,
		Endpoint:                    foreignPeer.Endpoint,
		AllowedIps:                  foreignPeer.AllowedIPs,
		PersistentKeepAliveInterval: int(foreignPeer.PersistentKeepaliveInterval),
		LastHandshakeTime:           adapt.ToPointer(foreignPeer.LastHandshakeTime),
		ReceiveBytes:                float64(foreignPeer.ReceiveBytes),
		TransmitBytes:               float64(foreignPeer.TransmitBytes),
		ProtocolVersion:             foreignPeer.ProtocolVersion,
	}
}
