package server

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var namePattern = regexp.MustCompile("[a-zA-Z0-9.-_]{1,16}")

type Server struct {
	Id           string
	Name         string
	Description  string
	Enabled      bool
	Running      bool
	PublicKey    string
	PrivateKey   string
	ListenPort   *int
	FirewallMark *int
	Address      string
	DNS          []string
	MTU          int
	Stats        Stats
	Hooks        []*Hook
	CreateUserId string
	UpdateUserId string
	DeleteUserId string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (s *Server) validate(fieldMask *UpdateFieldMask) error {
	if fieldMask == nil {
		if len(strings.TrimSpace(s.Name)) == 0 {
			return ErrNameRequired
		}

		if !namePattern.MatchString(s.Name) {
			return ErrInvalidName
		}
	}

	if fieldMask == nil || fieldMask.Description {
		if len(s.Name) > 255 {
			return fmt.Errorf("description must not be longer than 255 characters")
		}
	}

	if fieldMask == nil || fieldMask.PrivateKey {
		if len(strings.TrimSpace(s.PrivateKey)) == 0 {
			if _, err := wgtypes.GeneratePrivateKey(); err != nil {
				return fmt.Errorf("failed to generate private key: %w", err)
			}
		}
	}

	if fieldMask == nil || fieldMask.ListenPort {
		if s.ListenPort != nil && (*s.ListenPort < 1 || *s.ListenPort > 65535) {
			return fmt.Errorf("invalid listen port: %d", *s.ListenPort)
		}
	}

	if fieldMask == nil || fieldMask.FirewallMark {
		if s.FirewallMark != nil && (*s.FirewallMark < 1 || *s.FirewallMark > math.MaxInt32) {
			return fmt.Errorf("invalid firewall mark: %d", *s.FirewallMark)
		}
	}

	if fieldMask == nil || fieldMask.Address {
		if _, _, err := net.ParseCIDR(s.Address); err != nil {
			return fmt.Errorf("invalid address: %s", s.Address)
		}
	}

	if fieldMask == nil || fieldMask.DNS {
		for i, dns := range s.DNS {
			if net.ParseIP(dns) == nil {
				return fmt.Errorf("invalid dns ip address %d: %s", i+1, dns)
			}
		}
	}

	if fieldMask == nil || fieldMask.MTU {
		if s.MTU < 1280 || s.MTU > 1500 {
			return ErrInvalidMtu
		}
	}

	if fieldMask == nil || fieldMask.Hooks {
		for i, hook := range s.Hooks {
			if _, err := exec.LookPath(hook.Command); err != nil {
				return fmt.Errorf("invalid  server hook #%d command: %s - %w", i+1, hook.Command, err)
			}
			if i != len(s.Hooks)-1 {
				for _, nextHook := range s.Hooks[i+1:] {
					if strings.EqualFold(hook.Command, nextHook.Command) {
						return fmt.Errorf("hook command: %s already exists", hook.Command)
					}
				}
			}
		}
	}

	return nil
}
func (s *Server) update(options *UpdateOptions, fieldMask *UpdateFieldMask) {
	if fieldMask.Description {
		s.Description = options.Description
	}

	if fieldMask.Enabled {
		s.Enabled = options.Enabled
	}

	if fieldMask.Running {
		s.Running = options.Running
	}

	if fieldMask.PublicKey {
		s.PublicKey = options.PublicKey
	}

	if fieldMask.ListenPort {
		s.ListenPort = options.ListenPort
	}

	if fieldMask.FirewallMark {
		s.FirewallMark = options.FirewallMark
	}

	if fieldMask.Address {
		s.Address = options.Address
	}

	if fieldMask.DNS {
		s.DNS = options.DNS
	}

	if fieldMask.MTU {
		s.MTU = options.MTU
	}

	if fieldMask.Stats {
		s.Stats = options.Stats
	}

	if fieldMask.Hooks {
		s.Hooks = options.Hooks
	}
}

func (s *Server) runHooks(action HookAction) error {
	var errs []error
	for i, hook := range s.Hooks {
		if !hook.ShouldExecute(action) {
			continue
		}

		cmd := exec.Command(hook.Command, "SERVER", s.Name, string(action))
		cmd.Env = []string{
			fmt.Sprintf("WG_SERVER_NAME=%s", s.Name),
			fmt.Sprintf("WG_SERVER_DESCRIPTION=%s", strings.ReplaceAll(s.Description, "\n", "\\n")),
			fmt.Sprintf("WG_SERVER_PUBLICKEY=%s", s.PublicKey),
			fmt.Sprintf("WG_SERVER_LISTEN_PORT=%d", adapt.Dereference(s.ListenPort)),
			fmt.Sprintf("WG_SERVER_FIREWALL_MARK=%d", adapt.Dereference(s.FirewallMark)),
			fmt.Sprintf("WG_SERVER_ADDRESS=%s", s.Address),
			fmt.Sprintf("WG_SERVER_DNS=%s", strings.Join(s.DNS, ",")),
			fmt.Sprintf("WG_SERVER_MTU=%d", s.MTU),
		}
		if err := cmd.Run(); err != nil {
			errs = append(errs, fmt.Errorf("failed to execute hook #%d %s - %w", i+1, hook.Command, err))
		}
	}
	return errors.Join(errs...)
}
