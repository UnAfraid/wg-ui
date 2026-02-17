package backend

import (
	"fmt"
	"sync"
)

// Factory is a function that creates a new Backend instance
type Factory func() (Backend, error)

// Registration holds factory and support info for a backend type
type Registration struct {
	Factory   Factory
	Supported bool
}

var (
	registryMu  sync.RWMutex
	registryMap = make(map[string]*Registration)
)

// Register registers a backend type with its factory and platform support status.
// This should be called from init() functions in backend implementations.
func Register(scheme string, factory Factory, supported bool) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registryMap[scheme] = &Registration{
		Factory:   factory,
		Supported: supported,
	}
}

// Get returns the registration for a backend type
func Get(scheme string) (*Registration, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	reg, ok := registryMap[scheme]
	return reg, ok
}

// IsSupported checks if a backend type is supported on the current platform
func IsSupported(scheme string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	reg, ok := registryMap[scheme]
	if !ok {
		return false
	}
	return reg.Supported
}

// ListTypes returns all registered backend type names
func ListTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	types := make([]string, 0, len(registryMap))
	for t := range registryMap {
		types = append(types, t)
	}
	return types
}

// ListSupportedTypes returns backend types supported on the current platform
func ListSupportedTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var types []string
	for t, reg := range registryMap {
		if reg.Supported {
			types = append(types, t)
		}
	}
	return types
}

// Create creates a new backend instance for the given scheme
func Create(scheme string) (Backend, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	reg, ok := registryMap[scheme]

	if !ok {
		return nil, fmt.Errorf("unknown backend type: %s", scheme)
	}

	if !reg.Supported {
		return nil, fmt.Errorf("backend type %s is not supported on this platform", scheme)
	}

	if reg.Factory == nil {
		return nil, fmt.Errorf("backend type %s has no factory registered", scheme)
	}

	return reg.Factory()
}
