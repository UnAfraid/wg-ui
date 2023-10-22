//go:build linux

package linux

import (
	"github.com/vishvananda/netlink"
)

type wgLink struct {
	attrs *netlink.LinkAttrs
}

func (w *wgLink) Attrs() *netlink.LinkAttrs {
	return w.attrs
}

func (w *wgLink) Type() string {
	return "wireguard"
}
