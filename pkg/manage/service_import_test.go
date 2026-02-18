package manage

import (
	"strings"
	"testing"
)

func TestImportedPeerNameUsesPreferredName(t *testing.T) {
	used := map[string]struct{}{}

	got := importedPeerName("Office Laptop", 0, used)
	if got != "Office Laptop" {
		t.Fatalf("expected preferred name to be used, got %q", got)
	}
}

func TestImportedPeerNameFallsBackToDefault(t *testing.T) {
	used := map[string]struct{}{}

	got := importedPeerName("ab", 0, used)
	if got != "Peer #1" {
		t.Fatalf("expected fallback name, got %q", got)
	}
}

func TestImportedPeerNameEnsuresCaseInsensitiveUniqueness(t *testing.T) {
	used := map[string]struct{}{}

	first := importedPeerName("Office Laptop", 0, used)
	second := importedPeerName("office laptop", 1, used)

	if first != "Office Laptop" {
		t.Fatalf("unexpected first name %q", first)
	}
	if second != "office laptop (2)" {
		t.Fatalf("unexpected second name %q", second)
	}
}

func TestImportedPeerNameAppliesLengthLimitWithSuffix(t *testing.T) {
	used := map[string]struct{}{}
	base := "123456789012345678901234567890"

	first := importedPeerName(base, 0, used)
	second := importedPeerName(base, 1, used)

	if first != base {
		t.Fatalf("expected first name %q, got %q", base, first)
	}
	if !strings.HasSuffix(second, " (2)") {
		t.Fatalf("expected suffixed duplicate name, got %q", second)
	}
	if len([]rune(second)) > 30 {
		t.Fatalf("expected name length <= 30, got %d for %q", len([]rune(second)), second)
	}
}
