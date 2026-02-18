package model

import (
	"testing"
	"time"

	"github.com/UnAfraid/wg-ui/pkg/backend"
)

func TestToBackendRedactsURLPassword(t *testing.T) {
	now := time.Now()
	mapped := ToBackend(&backend.Backend{
		Id:          "backend-id",
		Name:        "router",
		Description: "desc",
		Url:         "routeros://admin:secret@router.example.com:443/rest",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	if mapped == nil {
		t.Fatalf("expected mapped backend")
	}

	expected := "routeros://admin:%2A%2A%2A@router.example.com:443/rest"
	if mapped.URL != expected {
		t.Fatalf("expected redacted url %q, got %q", expected, mapped.URL)
	}
}
