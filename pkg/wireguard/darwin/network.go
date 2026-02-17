//go:build darwin

package darwin

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

// configureInterface sets up the network interface with IP address and MTU
func configureInterface(name string, address string, mtu int) error {
	// Parse CIDR address to validate it and extract IP
	ip, ipNet, err := net.ParseCIDR(address)
	if err != nil {
		return fmt.Errorf("invalid CIDR address: %w", err)
	}

	// Convert netmask to dotted decimal format
	mask := fmt.Sprintf("%d.%d.%d.%d", ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])

	// utun interfaces on macOS are point-to-point interfaces and require:
	// ifconfig <interface> <local_addr> <dest_addr> netmask <mask>
	// For WireGuard, we use the same address for both local and destination
	// This creates a point-to-point link where the interface acts as both ends
	localAddr := ip.String()
	destAddr := ip.String()

	// Configure the interface with local, destination, and netmask
	cmd := exec.Command("ifconfig", name, "inet", localAddr, destAddr, "netmask", mask)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %w (output: %s)", err, string(output))
	}

	// Set MTU
	cmd = exec.Command("ifconfig", name, "mtu", fmt.Sprintf("%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %w (output: %s)", err, string(output))
	}

	// Bring interface up
	cmd = exec.Command("ifconfig", name, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w (output: %s)", err, string(output))
	}

	return nil
}

// configureRoutes adds routes for peer allowed IPs using the route command
func configureRoutes(ifName string, peers []*backend.PeerOptions) error {
	for _, peer := range peers {
		for _, allowedIPStr := range peer.AllowedIPs {
			_, ipNet, err := net.ParseCIDR(allowedIPStr)
			if err != nil {
				return fmt.Errorf("failed to parse allowed IP %s: %w", allowedIPStr, err)
			}

			// Use route command: route -n add -net <network>/<prefix> -interface <ifname>
			ones, _ := ipNet.Mask.Size()
			dest := fmt.Sprintf("%s/%d", ipNet.IP.String(), ones)

			cmd := exec.Command("route", "-n", "add", "-net", dest, "-interface", ifName)
			if output, err := cmd.CombinedOutput(); err != nil {
				outStr := strings.TrimSpace(string(output))
				// "File exists" means route already exists, which is fine
				if !strings.Contains(outStr, "File exists") {
					return fmt.Errorf("failed to add route for %s: %w (output: %s)", ipNet.String(), err, outStr)
				}
			}
		}
	}

	return nil
}

// removeRoutes removes all routes associated with the interface
func removeRoutes(ifName string) error {
	// Get all routes from the routing table
	rib, err := route.FetchRIB(unix.AF_UNSPEC, unix.NET_RT_DUMP, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch routing table: %w", err)
	}

	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return nil // interface already gone
	}

	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return fmt.Errorf("failed to parse routing table: %w", err)
	}

	// Delete routes that belong to our interface
	for _, msg := range msgs {
		rtMsg, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}
		if rtMsg.Index == iface.Index {
			deleteRoute(rtMsg)
		}
	}

	return nil
}

// deleteRoute removes a route from the routing table
func deleteRoute(rtMsg *route.RouteMessage) error {
	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	rtMsg.Type = unix.RTM_DELETE
	rtMsg.ID = uintptr(os.Getpid())
	rtMsg.Seq++

	b, err := rtMsg.Marshal()
	if err != nil {
		return err
	}

	_, err = unix.Write(fd, b)
	return err
}

// interfaceStats retrieves interface statistics
func interfaceStats(name string) (*backend.InterfaceStats, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	// On macOS, we can't easily get detailed interface statistics from net package
	// We would need to use syscall to get detailed stats
	// For now, return basic stats based on what we can get from WireGuard peers

	// Get WireGuard device stats by summing peer stats
	// This is handled by the caller which has access to wgctrl

	_ = iface // Placeholder for when we implement detailed stats

	return &backend.InterfaceStats{
		RxBytes: 0,
		TxBytes: 0,
	}, nil
}
