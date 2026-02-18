package routeros

import "testing"

func TestParseURLDefaults(t *testing.T) {
	parsed, err := parseURL("routeros://admin:secret@192.168.88.1/")
	if err != nil {
		t.Fatalf("parseURL returned error: %v", err)
	}

	if parsed.baseURL != "https://192.168.88.1:443/rest" {
		t.Fatalf("unexpected baseURL: %q", parsed.baseURL)
	}
	if parsed.username != "admin" {
		t.Fatalf("unexpected username: %q", parsed.username)
	}
	if parsed.password != "secret" {
		t.Fatalf("unexpected password: %q", parsed.password)
	}
	if parsed.insecureSkipVerify {
		t.Fatalf("expected insecureSkipVerify to be disabled by default")
	}
}

func TestParseURLWithInsecureSkipVerify(t *testing.T) {
	parsed, err := parseURL("routeros://api:pw@router.example.com:8443/?insecureSkipVerify=true")
	if err != nil {
		t.Fatalf("parseURL returned error: %v", err)
	}

	if parsed.baseURL != "https://router.example.com:8443/rest" {
		t.Fatalf("unexpected baseURL: %q", parsed.baseURL)
	}
	if !parsed.insecureSkipVerify {
		t.Fatalf("expected insecureSkipVerify=true")
	}
}

func TestParseURLRejectsHTTP(t *testing.T) {
	if _, err := parseURL("routeros://api:pw@router.example.com/?https=false"); err == nil {
		t.Fatalf("expected https=false to fail")
	}
}

func TestParseURLRequiresCredentials(t *testing.T) {
	if _, err := parseURL("routeros://admin@192.168.88.1/"); err == nil {
		t.Fatalf("expected missing password to fail")
	}
}
