package routeros

import (
	"testing"
	"time"
)

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

func TestShouldSkipForeignInterface(t *testing.T) {
	tests := []struct {
		name        string
		iface       entry
		peerEntries []entry
		want        bool
	}{
		{
			name:  "dynamic interface",
			iface: entry{"name": "wg0", "dynamic": "true"},
			want:  true,
		},
		{
			name:  "back-to-home name",
			iface: entry{"name": "back-to-home-vpn"},
			want:  true,
		},
		{
			name:  "back to home comment",
			iface: entry{"name": "wg0", "comment": "Back to Home"},
			want:  true,
		},
		{
			name:        "dynamic peer",
			iface:       entry{"name": "wg0"},
			peerEntries: []entry{{"public-key": "abc", "dynamic": "true"}},
			want:        true,
		},
		{
			name:        "regular interface",
			iface:       entry{"name": "wg0"},
			peerEntries: []entry{{"public-key": "abc"}},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipForeignInterface(tt.iface, tt.peerEntries)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestPeerNeedsPatch(t *testing.T) {
	existing := entry{
		"name":                 "peer-1",
		"comment":              "Office laptop",
		"public-key":           "pubkey",
		"allowed-address":      "10.0.0.2/32, 10.0.0.3/32",
		"endpoint-address":     "vpn.example.com",
		"endpoint-port":        "51820",
		"persistent-keepalive": "25",
		"preshared-key":        "psk",
		"disabled":             "false",
	}

	desiredSame := map[string]string{
		"name":                 "peer-1",
		"comment":              "Office laptop",
		"public-key":           "pubkey",
		"allowed-address":      "10.0.0.3/32,10.0.0.2/32",
		"endpoint-address":     "vpn.example.com",
		"endpoint-port":        "51820",
		"persistent-keepalive": "25",
		"preshared-key":        "psk",
		"disabled":             "false",
	}
	if peerNeedsPatch(existing, desiredSame) {
		t.Fatalf("expected no patch for equivalent peer settings")
	}

	desiredChanged := map[string]string{
		"name":                 "peer-1",
		"comment":              "Office laptop",
		"public-key":           "pubkey",
		"allowed-address":      "10.0.0.4/32",
		"endpoint-address":     "vpn.example.com",
		"endpoint-port":        "51820",
		"persistent-keepalive": "25",
		"preshared-key":        "psk",
		"disabled":             "false",
	}
	if !peerNeedsPatch(existing, desiredChanged) {
		t.Fatalf("expected patch when allowed-address changes")
	}

	desiredRenamed := map[string]string{
		"name":                 "peer-renamed",
		"comment":              "Office laptop",
		"public-key":           "pubkey",
		"allowed-address":      "10.0.0.3/32,10.0.0.2/32",
		"endpoint-address":     "vpn.example.com",
		"endpoint-port":        "51820",
		"persistent-keepalive": "25",
		"preshared-key":        "psk",
		"disabled":             "false",
	}
	if !peerNeedsPatch(existing, desiredRenamed) {
		t.Fatalf("expected patch when peer name changes")
	}
}

func TestInterfaceNeedsPatch(t *testing.T) {
	existing := entry{
		"name":        "wg0",
		"private-key": "private-key",
		"comment":     "Office",
		"mtu":         "1420",
		"listen-port": "51820",
		"disabled":    "false",
	}

	desiredSame := map[string]string{
		"name":        "wg0",
		"private-key": "private-key",
		"comment":     "Office",
		"mtu":         "1420",
		"listen-port": "51820",
		"disabled":    "false",
	}
	if interfaceNeedsPatch(existing, desiredSame) {
		t.Fatalf("expected no patch for equivalent interface settings")
	}

	desiredChanged := map[string]string{
		"name":        "wg0",
		"private-key": "private-key",
		"comment":     "Home",
		"mtu":         "1420",
		"listen-port": "51820",
		"disabled":    "false",
	}
	if !interfaceNeedsPatch(existing, desiredChanged) {
		t.Fatalf("expected patch when interface comment changes")
	}
}

func TestParseRouterOSHandshakeTimeDuration(t *testing.T) {
	now := time.Date(2026, 2, 18, 16, 40, 0, 0, time.UTC)
	handshake := parseRouterOSHandshakeTime("1m29s", now)

	expected := now.Add(-89 * time.Second)
	if !handshake.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, handshake)
	}
}

func TestPeerStatsFromEntryParsesHumanReadableCounters(t *testing.T) {
	stats := peerStatsFromEntry(entry{
		"last-handshake": "1m29s",
		"rx":             "2450.9KiB",
		"tx":             "42.3MiB",
	})

	if stats.LastHandshakeTime.IsZero() {
		t.Fatalf("expected non-zero last handshake")
	}
	if stats.ReceiveBytes == 0 {
		t.Fatalf("expected non-zero receive bytes")
	}
	if stats.TransmitBytes == 0 {
		t.Fatalf("expected non-zero transmit bytes")
	}
}

func TestParseRouterOSByteSize(t *testing.T) {
	tests := []struct {
		raw      string
		expected uint64
	}{
		{raw: "2450.9KiB", expected: 2509722},
		{raw: "42.3MiB", expected: 44354765},
		{raw: "123", expected: 123},
	}

	for _, tt := range tests {
		got, err := parseRouterOSByteSize(tt.raw)
		if err != nil {
			t.Fatalf("parseRouterOSByteSize(%q) returned error: %v", tt.raw, err)
		}
		if got != tt.expected {
			t.Fatalf("parseRouterOSByteSize(%q) expected %d, got %d", tt.raw, tt.expected, got)
		}
	}
}

func TestEndpointValuePrefersCurrentEndpoint(t *testing.T) {
	got := endpointValue(entry{
		"endpoint-address":         "",
		"endpoint-port":            "0",
		"current-endpoint-address": "62.176.113.208",
		"current-endpoint-port":    "46138",
	})

	if got != "62.176.113.208:46138" {
		t.Fatalf("expected current endpoint, got %q", got)
	}
}

func TestEndpointValueFallsBackToConfiguredEndpoint(t *testing.T) {
	got := endpointValue(entry{
		"endpoint-address":         "vpn.example.com",
		"endpoint-port":            "51820",
		"current-endpoint-address": "",
		"current-endpoint-port":    "0",
	})

	if got != "vpn.example.com:51820" {
		t.Fatalf("expected configured endpoint fallback, got %q", got)
	}
}

func TestEndpointValueFormatsIPv6CurrentEndpoint(t *testing.T) {
	got := endpointValue(entry{
		"current-endpoint-address": "2001:db8::1",
		"current-endpoint-port":    "51820",
	})

	if got != "[2001:db8::1]:51820" {
		t.Fatalf("expected IPv6 endpoint to be bracketed, got %q", got)
	}
}
