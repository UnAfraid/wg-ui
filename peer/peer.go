package peer

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Peer struct {
	Id                  string
	ServerId            string
	Name                string
	Description         string
	PublicKey           string
	Endpoint            string
	AllowedIPs          []string
	PresharedKey        string
	PersistentKeepalive int
	Hooks               []*Hook
	CreateUserId        string
	UpdateUserId        string
	DeleteUserId        string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}

func (p *Peer) validate(fieldMask *UpdateFieldMask) error {
	if fieldMask == nil || fieldMask.Name {
		if len(strings.TrimSpace(p.Name)) == 0 {
			return ErrNameRequired
		}

		if len(p.Name) < 3 {
			return fmt.Errorf("name must be at least 3 characters long")
		}
		if len(p.Name) > 30 {
			return fmt.Errorf("name must not be longer than 30 characters")
		}
	}

	if fieldMask == nil || fieldMask.Description {
		if len(p.Name) > 255 {
			return fmt.Errorf("description must not be longer than 255 characters")
		}
	}

	if fieldMask == nil || fieldMask.PublicKey {
		if len(strings.TrimSpace(p.PublicKey)) == 0 {
			return ErrPublicKeyRequired
		}
		if _, err := wgtypes.ParseKey(p.PublicKey); err != nil {
			return fmt.Errorf("invalid public key: %w", err)
		}
	}

	if fieldMask == nil || fieldMask.Endpoint {
		_, err := netip.ParseAddrPort(p.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint: %w", err)
		}
	}

	if fieldMask == nil || fieldMask.AllowedIPs {
		for i, allowedIP := range p.AllowedIPs {
			_, _, err := net.ParseCIDR(allowedIP)
			if err != nil {
				return fmt.Errorf("invalid allowed ip address: %d - %w", i+1, err)
			}
		}
	}

	if fieldMask == nil || fieldMask.PresharedKey {
		if len(p.PresharedKey) != 0 {
			if _, err := wgtypes.ParseKey(p.PresharedKey); err != nil {
				return fmt.Errorf("invalid preshared key: %s", err)
			}
		}
	}

	if fieldMask == nil || fieldMask.PersistentKeepalive {
		if p.PersistentKeepalive != 0 && p.PersistentKeepalive < 0 || p.PersistentKeepalive > 65535 {
			return fmt.Errorf("invalid persistent keep alive: %d, expected value between 1 and 65535", p.PersistentKeepalive)
		}
	}

	if fieldMask == nil || fieldMask.Hooks {
		for i, hook := range p.Hooks {
			if _, err := exec.LookPath(hook.Command); err != nil {
				return fmt.Errorf("invalid peer hook #%d command: %s - %w", i+1, hook.Command, err)
			}

			if i != len(p.Hooks)-1 {
				for _, nextHook := range p.Hooks[i+1:] {
					if strings.EqualFold(hook.Command, nextHook.Command) {
						return fmt.Errorf("hook command: %s already exists", hook.Command)
					}
				}
			}
		}
	}

	return nil
}

func (p *Peer) update(options *UpdateOptions, fieldMask *UpdateFieldMask) {
	if fieldMask.Name {
		p.Name = options.Name
	}

	if fieldMask.Description {
		p.Description = options.Description
	}

	if fieldMask.PublicKey {
		p.PublicKey = options.PublicKey
	}

	if fieldMask.Endpoint {
		p.Endpoint = options.Endpoint
	}

	if fieldMask.AllowedIPs {
		p.AllowedIPs = options.AllowedIPs
	}

	if fieldMask.PresharedKey {
		p.PresharedKey = options.PresharedKey
	}

	if fieldMask.PersistentKeepalive {
		p.PersistentKeepalive = options.PersistentKeepalive
	}

	if fieldMask.Hooks {
		p.Hooks = options.Hooks
	}

	if fieldMask.CreateUserId {
		p.CreateUserId = options.CreateUserId
	}

	if fieldMask.UpdateUserId {
		p.UpdateUserId = options.UpdateUserId
	}
}

func (p *Peer) runHooks(action HookAction) error {
	var errs []error
	for i, hook := range p.Hooks {
		if !hook.shouldExecute(action) {
			continue
		}

		cmd := exec.Command(hook.Command, "PEER", p.PublicKey, string(action))
		cmd.Env = []string{
			fmt.Sprintf("WG_PEER_NAME=%s", p.Name),
			fmt.Sprintf("WG_PEER_DESCRIPTION=%s", strings.ReplaceAll(p.Description, "\n", "\\n")),
			fmt.Sprintf("WG_PEER_PUBLICKEY=%s", p.PublicKey),
			fmt.Sprintf("WG_PEER_ENDPOINT=%s", p.Endpoint),
			fmt.Sprintf("WG_PEER_ALLOWED_IPS=%s", strings.Join(p.AllowedIPs, ", ")),
			fmt.Sprintf("WG_PEER_PERSISTENT_KEEP_ALIVE=%d", p.PersistentKeepalive),
		}
		if err := cmd.Run(); err != nil {
			errs = append(errs, fmt.Errorf("failed to execute hook #%d %s - %w", i+1, hook.Command, err))
		}
	}
	return errors.Join(errs...)
}
