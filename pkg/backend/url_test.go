package backend

import "testing"

func TestParseURLExecBackend(t *testing.T) {
	parsed, err := ParseURL("exec:///etc/wireguard?sudo=true")
	if err != nil {
		t.Fatalf("ParseURL returned error: %v", err)
	}

	if parsed.Type != "exec" {
		t.Fatalf("expected type exec, got %q", parsed.Type)
	}
	if parsed.Path != "/etc/wireguard" {
		t.Fatalf("expected path /etc/wireguard, got %q", parsed.Path)
	}
	if parsed.Options["sudo"] != "true" {
		t.Fatalf("expected sudo=true option, got %q", parsed.Options["sudo"])
	}
}

func TestParseURLAllowsUnknownScheme(t *testing.T) {
	parsed, err := ParseURL("custom:///tmp/path")
	if err != nil {
		t.Fatalf("ParseURL returned error: %v", err)
	}

	if parsed.Type != "custom" {
		t.Fatalf("expected type custom, got %q", parsed.Type)
	}
}
