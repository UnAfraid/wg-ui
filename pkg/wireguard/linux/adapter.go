//go:build linux

package linux

import (
	"fmt"
	"net"
	"time"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

func wireguardPeerOptionsToPeerConfig(peer *driver.PeerOptions) (wgtypes.PeerConfig, error) {
	publicKey, err := wgtypes.ParseKey(peer.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, fmt.Errorf("invalid peer: %s public key: %w", peer.PublicKey, err)
	}

	var presharedKey *wgtypes.Key
	if peer.PresharedKey != "" {
		key, err := wgtypes.ParseKey(peer.PresharedKey)
		if err != nil {
			return wgtypes.PeerConfig{}, fmt.Errorf("invalid peer: %s preshared key - %w", peer.PublicKey, err)
		}
		presharedKey = &key
	}

	allowedIPs := make([]net.IPNet, len(peer.AllowedIPs))
	for i, cidr := range peer.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return wgtypes.PeerConfig{}, err
		}
		allowedIPs[i] = *ipNet
	}

	var persistentKeepaliveInterval *time.Duration
	if peer.PersistentKeepalive != 0 {
		persistentKeepaliveInterval = adapt.ToPointer(time.Duration(peer.PersistentKeepalive) * time.Second)
	}

	return wgtypes.PeerConfig{
		PublicKey:                   publicKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                presharedKey,
		PersistentKeepaliveInterval: persistentKeepaliveInterval,
		ReplaceAllowedIPs:           false,
		AllowedIPs:                  allowedIPs,
	}, nil
}

func linkStatisticsToBackendInterfaceStats(statistics *netlink.LinkStatistics) *driver.InterfaceStats {
	if statistics == nil {
		return nil
	}
	return &driver.InterfaceStats{
		RxBytes: statistics.RxBytes,
		TxBytes: statistics.TxBytes,
	}
}

func netlinkInterfaceToForeignInterface(link netlink.Link) (*driver.ForeignInterface, error) {
	attrs := link.Attrs()

	addrList, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to get address list for interface %s", attrs.Name)
	}

	var addresses []string
	for _, addr := range addrList {
		addresses = append(addresses, addr.IPNet.String())
	}

	return &driver.ForeignInterface{
		Name:      attrs.Name,
		Addresses: addresses,
		Mtu:       attrs.MTU,
		State:     attrs.OperState.String(),
	}, nil
}
