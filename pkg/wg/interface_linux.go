package wg

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/vishvananda/netlink"
)

func configureInterface(name string, address string, mtu int) error {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name

	link := wgLink{
		attrs: &attrs,
	}

	if err := netlink.LinkAdd(&link); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to add interface: %w", err)
		}
	}

	addressList, err := netlink.AddrList(&link, netFamilyAll)
	if err != nil {
		return fmt.Errorf("failed to get interface: %s address list: %w", name, err)
	}

	serverAddress, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("failed to parse client ip range: %w", err)
	}

	needsAddress := true
	for _, addr := range addressList {
		if addr.Equal(*serverAddress) {
			needsAddress = false
			break
		}
	}

	if needsAddress {
		if err = netlink.AddrAdd(&link, serverAddress); err != nil {
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add address: %w", err)
			}
		}
	}

	if mtu != attrs.MTU {
		if err = netlink.LinkSetMTU(&link, mtu); err != nil {
			return fmt.Errorf("failed to set server mtu: %w", err)
		}
	}

	if attrs.OperState != netlink.OperUp {
		if err = netlink.LinkSetUp(&link); err != nil {
			return fmt.Errorf("failed to set interface up: %w", err)
		}
	}

	return nil
}

func deleteInterface(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if os.IsNotExist(err) || errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}
		return fmt.Errorf("failed to find link by name: %w", err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface down: %w", err)
	}
	return nil
}

func interfaceStats(name string) (server.Stats, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if os.IsNotExist(err) || errors.As(err, &netlink.LinkNotFoundError{}) {
			return server.Stats{}, nil
		}
		return server.Stats{}, fmt.Errorf("failed to find link by name: %w", err)
	}

	statistics := link.Attrs().Statistics
	return server.Stats{
		RxPackets:         statistics.RxPackets,
		TxPackets:         statistics.TxPackets,
		RxBytes:           statistics.RxBytes,
		TxBytes:           statistics.TxBytes,
		RxErrors:          statistics.RxErrors,
		TxErrors:          statistics.TxErrors,
		RxDropped:         statistics.RxDropped,
		TxDropped:         statistics.TxDropped,
		Multicast:         statistics.Multicast,
		Collisions:        statistics.Collisions,
		RxLengthErrors:    statistics.RxLengthErrors,
		RxOverErrors:      statistics.RxOverErrors,
		RxCrcErrors:       statistics.RxCrcErrors,
		RxFrameErrors:     statistics.RxFrameErrors,
		RxFifoErrors:      statistics.RxFifoErrors,
		RxMissedErrors:    statistics.RxMissedErrors,
		TxAbortedErrors:   statistics.TxAbortedErrors,
		TxCarrierErrors:   statistics.TxCarrierErrors,
		TxFifoErrors:      statistics.TxFifoErrors,
		TxHeartbeatErrors: statistics.TxHeartbeatErrors,
		TxWindowErrors:    statistics.TxWindowErrors,
		RxCompressed:      statistics.RxCompressed,
		TxCompressed:      statistics.TxCompressed,
	}, nil
}

func findForeignInterfaces(knownInterfaces []string) (foreignInterfaces []ForeignInterface, err error) {
	list, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	for _, link := range list {
		if !strings.EqualFold(link.Type(), "wireguard") {
			continue
		}

		attrs := link.Attrs()
		name := attrs.Name
		if slices.Contains(knownInterfaces, name) {
			continue
		}

		addrList, err := netlink.AddrList(link, netFamilyAll)
		if err != nil {
			return nil, fmt.Errorf("failed to get address list for interface %s", name)
		}

		var addresses []string
		for _, addr := range addrList {
			addresses = append(addresses, addr.IPNet.String())
		}

		foreignInterfaces = append(foreignInterfaces, ForeignInterface{
			Name:      name,
			Addresses: addresses,
			Mtu:       attrs.MTU,
			State:     attrs.OperState.String(),
		})
	}
	return foreignInterfaces, nil
}
