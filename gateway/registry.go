package gateway

import (
	"fmt"
	"sync"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

// Factory constructs a payment gateway from provider-specific configuration.
type Factory func(config any) (paykit.Gateway, error)

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// Register adds a provider factory to the registry.
func Register(name string, factory Factory) {
	if name == "" {
		panic("gateway: provider name cannot be empty")
	}
	if factory == nil {
		panic("gateway: factory cannot be nil")
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := factories[name]; exists {
		panic(fmt.Sprintf("gateway: provider %q already registered", name))
	}

	factories[name] = factory
}

// New constructs a registered provider gateway.
func New(name string, config any) (paykit.Gateway, error) {
	mu.RLock()
	factory, ok := factories[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("gateway: unknown provider %q", name)
	}

	return factory(config)
}
