package wireguard

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

type testBackend struct {
	closeCalls int
}

func (t *testBackend) Device(context.Context, string) (*driver.Device, error) {
	return nil, nil
}

func (t *testBackend) Up(context.Context, driver.ConfigureOptions) (*driver.Device, error) {
	return nil, nil
}

func (t *testBackend) Down(context.Context, string) error {
	return nil
}

func (t *testBackend) Status(context.Context, string) (bool, error) {
	return false, nil
}

func (t *testBackend) Stats(context.Context, string) (*driver.InterfaceStats, error) {
	return nil, nil
}

func (t *testBackend) PeerStats(context.Context, string, string) (*driver.PeerStats, error) {
	return nil, nil
}

func (t *testBackend) FindForeignServers(context.Context, []string) ([]*driver.ForeignServer, error) {
	return nil, nil
}

func (t *testBackend) Close(context.Context) error {
	t.closeCalls++
	return nil
}

func TestRegistryRecreatesBackendWhenURLChanges(t *testing.T) {
	scheme := fmt.Sprintf("registry-test-%d", time.Now().UnixNano())
	created := make(map[string]*testBackend)

	driver.Register(scheme, func(_ context.Context, rawURL string) (driver.Backend, error) {
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

func TestRegistryGetOrCreateConcurrentSingleflight(t *testing.T) {
	scheme := fmt.Sprintf("registry-concurrent-%d", time.Now().UnixNano())
	var createCalls int32

	driver.Register(scheme, func(_ context.Context, rawURL string) (driver.Backend, error) {
		atomic.AddInt32(&createCalls, 1)
		time.Sleep(20 * time.Millisecond)
		return &testBackend{}, nil
	}, true)

	registry := NewRegistry()
	ctx := context.Background()

	const workers = 32
	results := make([]driver.Backend, workers)
	errs := make([]error, workers)

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)

	for i := 0; i < workers; i++ {
		go func(index int) {
			defer waitGroup.Done()
			<-start
			results[index], errs[index] = registry.GetOrCreate(ctx, "backend-id", scheme, "scheme:///same")
		}(i)
	}

	close(start)
	waitGroup.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d returned error: %v", i, err)
		}
	}

	first := results[0]
	if first == nil {
		t.Fatalf("expected backend instance, got nil")
	}
	for i := 1; i < workers; i++ {
		if results[i] != first {
			t.Fatalf("expected all workers to receive the same backend instance")
		}
	}

	if calls := atomic.LoadInt32(&createCalls); calls != 1 {
		t.Fatalf("expected 1 backend creation, got %d", calls)
	}
}
