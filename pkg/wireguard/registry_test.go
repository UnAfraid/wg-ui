package wireguard

import (
	"context"
	"fmt"
	"testing"
	"time"

	wgbackend "github.com/UnAfraid/wg-ui/pkg/wireguard/backend"
)

type testBackend struct {
	closeCalls int
}

func (t *testBackend) Device(context.Context, string) (*wgbackend.Device, error) {
	return nil, nil
}

func (t *testBackend) Up(context.Context, wgbackend.ConfigureOptions) (*wgbackend.Device, error) {
	return nil, nil
}

func (t *testBackend) Down(context.Context, string) error {
	return nil
}

func (t *testBackend) Status(context.Context, string) (bool, error) {
	return false, nil
}

func (t *testBackend) Stats(context.Context, string) (*wgbackend.InterfaceStats, error) {
	return nil, nil
}

func (t *testBackend) PeerStats(context.Context, string, string) (*wgbackend.PeerStats, error) {
	return nil, nil
}

func (t *testBackend) FindForeignServers(context.Context, []string) ([]*wgbackend.ForeignServer, error) {
	return nil, nil
}

func (t *testBackend) Close(context.Context) error {
	t.closeCalls++
	return nil
}

func (t *testBackend) Supported() bool {
	return true
}

func TestRegistryRecreatesBackendWhenURLChanges(t *testing.T) {
	scheme := fmt.Sprintf("registry-test-%d", time.Now().UnixNano())
	created := make(map[string]*testBackend)

	wgbackend.Register(scheme, func(rawURL string) (wgbackend.Backend, error) {
		b := &testBackend{}
		created[rawURL] = b
		return b, nil
	}, true)

	registry := NewRegistry()
	ctx := context.Background()

	backend1, err := registry.GetOrCreate(ctx, "backend-id", scheme, "scheme:///first")
	if err != nil {
		t.Fatalf("GetOrCreate first failed: %v", err)
	}

	backend2, err := registry.GetOrCreate(ctx, "backend-id", scheme, "scheme:///first")
	if err != nil {
		t.Fatalf("GetOrCreate second failed: %v", err)
	}
	if backend1 != backend2 {
		t.Fatalf("expected same backend instance for same url")
	}

	backend3, err := registry.GetOrCreate(ctx, "backend-id", scheme, "scheme:///second")
	if err != nil {
		t.Fatalf("GetOrCreate with changed url failed: %v", err)
	}
	if backend3 == backend1 {
		t.Fatalf("expected a new backend instance when url changes")
	}

	oldBackend := created["scheme:///first"]
	if oldBackend == nil {
		t.Fatalf("expected old backend instance to exist in test map")
	}
	if oldBackend.closeCalls != 1 {
		t.Fatalf("expected old backend close to be called once, got %d", oldBackend.closeCalls)
	}
}
