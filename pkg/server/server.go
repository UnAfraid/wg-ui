package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
)

var namePattern = regexp.MustCompile("[a-zA-Z0-9.-_]{1,16}")

type Server struct {
	Id           string
	Name         string
	Description  string
	BackendId    string
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
		// WireGuard allows fwmark 0 ("off"), so only negative values are invalid.
		if s.FirewallMark != nil && (*s.FirewallMark < 0 || *s.FirewallMark > math.MaxInt32) {
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
			command := strings.TrimSpace(hook.Command)
			if command == "" {
				return fmt.Errorf("invalid server hook #%d command: command is required", i+1)
			}
			if strings.Contains(command, "\n") {
				return fmt.Errorf("invalid server hook #%d command: multiline commands are not supported", i+1)
			}

			if !(hook.RunOnPreUp ||
				hook.RunOnPostUp ||
				hook.RunOnPreDown ||
				hook.RunOnPostDown ||
				hook.RunOnStart ||
				hook.RunOnStop) {
				return fmt.Errorf("invalid server hook #%d: no lifecycle events selected", i+1)
			}
		}
	}

	return nil
}
func (s *Server) update(options *UpdateOptions, fieldMask *UpdateFieldMask) error {
	if fieldMask.Description {
		s.Description = options.Description
	}

	if fieldMask.BackendId {
		s.BackendId = options.BackendId
	}

	if fieldMask.Enabled {
		s.Enabled = options.Enabled
	}

	if fieldMask.Running {
		s.Running = options.Running
	}

	if fieldMask.PrivateKey {
		s.PrivateKey = options.PrivateKey
		key, err := wgtypes.ParseKey(s.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		s.PublicKey = key.PublicKey().String()
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

	if fieldMask.CreateUserId {
		s.CreateUserId = options.CreateUserId
	}

	if fieldMask.UpdateUserId {
		s.UpdateUserId = options.UpdateUserId
	}

	return nil
}

func (s *Server) runHooks(ctx context.Context, action HookAction) error {
	var errs []error
	for i, hook := range s.Hooks {
		if !hook.shouldExecute(action) {
			continue
		}

		command := interpolateHookCommand(hook.Command, s.Name)
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("WG_SERVER_NAME=%s", s.Name),
			fmt.Sprintf("WG_SERVER_DESCRIPTION=%s", strings.ReplaceAll(s.Description, "\n", "\\n")),
			fmt.Sprintf("WG_SERVER_PUBLICKEY=%s", s.PublicKey),
			fmt.Sprintf("WG_SERVER_LISTEN_PORT=%d", adapt.Dereference(s.ListenPort)),
			fmt.Sprintf("WG_SERVER_FIREWALL_MARK=%d", adapt.Dereference(s.FirewallMark)),
			fmt.Sprintf("WG_SERVER_ADDRESS=%s", s.Address),
			fmt.Sprintf("WG_SERVER_DNS=%s", strings.Join(s.DNS, ",")),
			fmt.Sprintf("WG_SERVER_MTU=%d", s.MTU),
			fmt.Sprintf("WG_SERVER_HOOK_ACTION=%s", string(action)),
		)
		if err := cmd.Run(); err != nil {
			errs = append(errs, fmt.Errorf("failed to execute hook #%d %q - %w", i+1, hook.Command, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Server) RunHooks(ctx context.Context, action HookAction) error {
	return s.runHooks(ctx, action)
}

func interpolateHookCommand(command string, interfaceName string) string {
	return strings.ReplaceAll(command, "%i", interfaceName)
}
