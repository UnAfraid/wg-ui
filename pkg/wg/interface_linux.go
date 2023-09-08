package wg

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"

	"github.com/UnAfraid/wg-ui/pkg/server"
	"github.com/sirupsen/logrus"
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

func configureRoutes(name string, allowedIPs []net.IPNet) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if os.IsNotExist(err) || errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}
		return fmt.Errorf("failed to find link by name: %w", err)
	}

	routes, err := netlink.RouteList(link, netFamilyAll)
	if err != nil {
		return fmt.Errorf("failed to get routes: %w", err)
	}

	routesToAdd, routesToUpdate, routesToRemove := computeRoutes(link, routes, allowedIPs)

	for i, route := range routesToAdd {
		if err = netlink.RouteAdd(routesToAdd[i]); err != nil {
			return fmt.Errorf("failed to add route for %s - %w", route.Dst.String(), err)
		}

		logrus.
			WithField("name", link.Attrs().Name).
			WithField("route", route.Dst.String()).
			Debug("route added")
	}

	for i, route := range routesToUpdate {
		if err = netlink.RouteReplace(routesToAdd[i]); err != nil {
			return fmt.Errorf("failed to replace route for %s - %w", route.Dst.String(), err)
		}

		logrus.
			WithField("name", link.Attrs().Name).
			WithField("route", route.Dst.String()).
			Debug("route replaced")
	}

	for i, route := range routesToRemove {
		if err = netlink.RouteDel(routesToAdd[i]); err != nil {
			return fmt.Errorf("failed to delete route for %s - %w", route.Dst.String(), err)
		}

		logrus.
			WithField("name", link.Attrs().Name).
			WithField("route", route.Dst.String()).
			Debug("route deleted")
	}
	return nil
}

func computeRoutes(link netlink.Link, existingRoutes []netlink.Route, allowedIPs []net.IPNet) ([]*netlink.Route, []*netlink.Route, []*netlink.Route) {
	var routesToAdd []*netlink.Route
	var routesToUpdate []*netlink.Route
	var routesToRemove []*netlink.Route
	for i, allowedIP := range allowedIPs {
		var existingRoute *netlink.Route
		for _, route := range existingRoutes {
			if route.Dst != nil && route.Dst.IP.Equal(allowedIP.IP) && slices.Equal(route.Dst.Mask, allowedIP.Mask) {
				existingRoute = &existingRoutes[i]
				break
			}
		}
		if existingRoute != nil {
			var update bool
			if existingRoute.Scope != netlink.SCOPE_LINK {
				existingRoute.Scope = netlink.SCOPE_LINK
				update = true
			}

			if existingRoute.Protocol != netlink.RouteProtocol(3) {
				existingRoute.Protocol = netlink.RouteProtocol(3)
				update = true
			}

			if existingRoute.Type != 1 {
				existingRoute.Type = 1
				update = true
			}

			if update {
				routesToUpdate = append(routesToUpdate, existingRoute)
			}
			continue
		}

		routesToAdd = append(routesToAdd, &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       &allowedIP,
			Protocol:  netlink.RouteProtocol(3),
			Type:      1,
		})
	}

	for i, existingRoute := range existingRoutes {
		var exists bool
		for _, allowedIP := range allowedIPs {
			exists = existingRoute.Dst != nil && existingRoute.Dst.IP.Equal(allowedIP.IP) && slices.Equal(existingRoute.Dst.Mask, allowedIP.Mask)
			if exists {
				break
			}
		}
		if !exists {
			routesToRemove = append(routesToRemove, &existingRoutes[i])
		}
	}

	return routesToAdd, routesToUpdate, routesToRemove
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
