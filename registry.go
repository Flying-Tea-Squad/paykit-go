package paykit

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ErrUnknownGateway is returned by Get when a requested payment gateway
// has not been registered.
var ErrUnknownGateway = errors.New("paykit: unknown payment gateway")

// Factory constructs an initialized Gateway instance.
// Providers register a Factory func with Register during package initialization
// or at runtime.
type Factory func() (Gateway, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register registers a payment gateway factory under the given provider name.
// If Register is called twice with the same name, or if name is empty or
// factory is nil, it panics. This follows the standard Go driver registration
// pattern (e.g., database/sql.Register) to ensure configuration errors are
// detected immediately at application startup.
func Register(name string, factory Factory) {
	if name == "" {
		panic("paykit: provider name cannot be empty")
	}
	if factory == nil {
		panic("paykit: factory cannot be nil")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("paykit: provider %q already registered", name))
	}

	registry[name] = factory
}

// Get constructs and returns the payment gateway registered under name.
// If no gateway is registered with that name, it returns an error wrapping
// ErrUnknownGateway.
func Get(name string) (Gateway, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownGateway, name)
	}

	return factory()
}

// RegisteredGateways returns a sorted list of names of all currently registered
// payment gateways.
func RegisteredGateways() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
