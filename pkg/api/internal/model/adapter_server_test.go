package model

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
)

func TestCreateServerInputToCreateServerOptionsTreatsZeroFirewallMarkAsNil(t *testing.T) {
	firewallMark := 0
	input := CreateServerInput{
		Name:         "wg0",
		BackendID:    StringID(IdKindBackend, "backend-id"),
		Address:      "10.0.0.1/24",
		FirewallMark: graphql.OmittableOf(&firewallMark),
	}

	options, err := CreateServerInputToCreateServerOptions(input)
	if err != nil {
		t.Fatalf("CreateServerInputToCreateServerOptions returned error: %v", err)
	}

	if options.FirewallMark != nil {
		t.Fatalf("expected firewall mark to be nil when input is 0, got %v", *options.FirewallMark)
	}
}

func TestUpdateServerInputToUpdateOptionsTreatsZeroFirewallMarkAsNil(t *testing.T) {
	firewallMark := 0
	input := UpdateServerInput{
		ID:           StringID(IdKindServer, "server-id"),
		FirewallMark: graphql.OmittableOf(&firewallMark),
	}

	options, fieldMask, err := UpdateServerInputToUpdateOptionsAndUpdateFieldMask(input)
	if err != nil {
		t.Fatalf("UpdateServerInputToUpdateOptionsAndUpdateFieldMask returned error: %v", err)
	}

	if !fieldMask.FirewallMark {
		t.Fatalf("expected firewall mark field mask to be set")
	}
	if options.FirewallMark != nil {
		t.Fatalf("expected firewall mark to be nil when input is 0, got %v", *options.FirewallMark)
	}
}
