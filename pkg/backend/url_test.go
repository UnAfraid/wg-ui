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

func TestRedactURLPassword(t *testing.T) {
	redacted := RedactURLPassword("routeros://admin:secret@router.local:443/rest?insecureSkipVerify=true")
	expected := "routeros://admin:%2A%2A%2A@router.local:443/rest?insecureSkipVerify=true"
	if redacted != expected {
		t.Fatalf("expected %q, got %q", expected, redacted)
	}
}

func TestRedactURLPasswordWithoutPassword(t *testing.T) {
	raw := "ssh://root@example.com/etc/wireguard?sudo=true"
	redacted := RedactURLPassword(raw)
	if redacted != raw {
		t.Fatalf("expected url without password to remain unchanged, got %q", redacted)
	}
}

func TestReplaceRedactedURLPassword(t *testing.T) {
	updated, err := ReplaceRedactedURLPassword(
		"routeros://admin:***@router.example.com:443/rest",
		"routeros://admin:secret@router.example.com:443/rest",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "routeros://admin:secret@router.example.com:443/rest"
	if updated != expected {
		t.Fatalf("expected %q, got %q", expected, updated)
	}
}

func TestReplaceRedactedURLPasswordWithoutExistingPassword(t *testing.T) {
	_, err := ReplaceRedactedURLPassword(
		"routeros://admin:***@router.example.com:443/rest",
		"routeros://admin@router.example.com:443/rest",
	)
	if err == nil {
		t.Fatalf("expected error for redacted password without existing password")
	}
}

func TestReplaceRedactedURLPasswordWithDoubleEncodedMask(t *testing.T) {
	updated, err := ReplaceRedactedURLPassword(
		"routeros://admin:%252A%252A%252A@router.example.com:443/rest",
		"routeros://admin:secret@router.example.com:443/rest",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "routeros://admin:secret@router.example.com:443/rest"
	if updated != expected {
		t.Fatalf("expected %q, got %q", expected, updated)
	}
}
