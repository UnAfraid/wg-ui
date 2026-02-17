package wireguard

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/UnAfraid/wg-ui/pkg/wireguard/driver"
)

type retryBackendRef struct {
	id          string
	backendType string
	url         string
}

func (r *retryBackendRef) ID() string {
	return r.id
}

func (r *retryBackendRef) Type() string {
	return r.backendType
}

func (r *retryBackendRef) URL() string {
	return r.url
}

type retryBackend struct {
	deviceErr   error
	deviceName  string
	deviceCalls int
	closeCalls  int
}

func (b *retryBackend) Device(_ context.Context, name string) (*driver.Device, error) {
	b.deviceCalls++
	if b.deviceErr != nil {
		return nil, b.deviceErr
	}
	if b.deviceName != "" {
		name = b.deviceName
	}

	return &driver.Device{
		Wireguard: driver.Wireguard{
			Name: name,
		},
	}, nil
}

func (b *retryBackend) Up(context.Context, driver.ConfigureOptions) (*driver.Device, error) {
	return nil, nil
}

func (b *retryBackend) Down(context.Context, string) error {
	return nil
}

func (b *retryBackend) Status(context.Context, string) (bool, error) {
	return false, nil
}

func (b *retryBackend) Stats(context.Context, string) (*driver.InterfaceStats, error) {
	return nil, nil
}

func (b *retryBackend) PeerStats(context.Context, string, string) (*driver.PeerStats, error) {
	return nil, nil
}

func (b *retryBackend) FindForeignServers(context.Context, []string) ([]*driver.ForeignServer, error) {
	return nil, nil
}

func (b *retryBackend) Close(context.Context) error {
	b.closeCalls++
	return nil
}

func TestServiceRetriesOnStaleBackendConnection(t *testing.T) {
	scheme := fmt.Sprintf("service-retry-%d", time.Now().UnixNano())

	var created int
	var firstBackend *retryBackend
	var secondBackend *retryBackend

	driver.Register(scheme, func(_ context.Context, rawURL string) (driver.Backend, error) {
		created++
		backend := &retryBackend{}
		if created == 1 {
			backend.deviceErr = fmt.Errorf("%w: dbus: connection closed", driver.ErrConnectionStale)
			firstBackend = backend
		} else {
			secondBackend = backend
		}
		return backend, nil
	}, true)

	registry := NewRegistry()
	service := NewService(registry)
	ref := &retryBackendRef{
		id:          "backend-id",
		backendType: scheme,
		url:         scheme + ":///wireguard",
	}

	device, err := service.Device(context.Background(), ref, "wg0")
	if err != nil {
		t.Fatalf("Device returned error: %v", err)
	}
	if device == nil {
		t.Fatalf("Device returned nil device")
	}
	if device.Wireguard.Name != "wg0" {
		t.Fatalf("expected wg0 device, got %q", device.Wireguard.Name)
	}

	if created != 2 {
		t.Fatalf("expected 2 backend creations, got %d", created)
	}
	if firstBackend == nil || secondBackend == nil {
		t.Fatalf("expected both backend instances to be created")
	}
	if firstBackend.closeCalls != 1 {
		t.Fatalf("expected first backend to be closed once, got %d", firstBackend.closeCalls)
	}
	if firstBackend.deviceCalls != 1 {
		t.Fatalf("expected first backend Device call count 1, got %d", firstBackend.deviceCalls)
	}
	if secondBackend.deviceCalls != 1 {
		t.Fatalf("expected second backend Device call count 1, got %d", secondBackend.deviceCalls)
	}
}

func TestServiceDoesNotRetryOnRegularBackendError(t *testing.T) {
	scheme := fmt.Sprintf("service-no-retry-%d", time.Now().UnixNano())

	var created int
	regularErr := errors.New("regular backend error")

	driver.Register(scheme, func(_ context.Context, rawURL string) (driver.Backend, error) {
		created++
		return &retryBackend{deviceErr: regularErr}, nil
	}, true)

	registry := NewRegistry()
	service := NewService(registry)
	ref := &retryBackendRef{
		id:          "backend-id",
		backendType: scheme,
		url:         scheme + ":///wireguard",
	}

	_, err := service.Device(context.Background(), ref, "wg0")
	if !errors.Is(err, regularErr) {
		t.Fatalf("expected regular backend error, got %v", err)
	}

	if created != 1 {
		t.Fatalf("expected 1 backend creation, got %d", created)
	}
}
