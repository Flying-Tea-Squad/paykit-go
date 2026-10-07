package paykit_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGateway is a minimal Gateway implementation for registry tests.
type mockGateway struct {
	name string
}

func (m *mockGateway) Name() string { return m.name }
func (m *mockGateway) Charge(_ context.Context, _ *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	return nil, nil
}
func (m *mockGateway) QueryStatus(_ context.Context, _ *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	return nil, nil
}
func (m *mockGateway) Refund(_ context.Context, _ *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	return nil, nil
}

func TestRegister_Success(t *testing.T) {
	name := "test_provider_success"
	expected := &mockGateway{name: name}

	paykit.Register(name, func() (paykit.Gateway, error) {
		return expected, nil
	})

	gw, err := paykit.Get(name)
	require.NoError(t, err)
	assert.Equal(t, name, gw.Name())
	assert.Same(t, expected, gw)
}

func TestRegister_Panics(t *testing.T) {
	t.Run("empty name panics", func(t *testing.T) {
		assert.PanicsWithValue(t, "paykit: provider name cannot be empty", func() {
			paykit.Register("", func() (paykit.Gateway, error) {
				return &mockGateway{name: "empty"}, nil
			})
		})
	})

	t.Run("nil factory panics", func(t *testing.T) {
		assert.PanicsWithValue(t, "paykit: factory cannot be nil", func() {
			paykit.Register("test_nil_factory", nil)
		})
	})

	t.Run("duplicate registration panics", func(t *testing.T) {
		name := "test_duplicate_provider"
		paykit.Register(name, func() (paykit.Gateway, error) {
			return &mockGateway{name: name}, nil
		})

		assert.PanicsWithValue(t, fmt.Sprintf("paykit: provider %q already registered", name), func() {
			paykit.Register(name, func() (paykit.Gateway, error) {
				return &mockGateway{name: name}, nil
			})
		})
	})
}

func TestGet_UnknownGateway(t *testing.T) {
	gw, err := paykit.Get("non_existent_provider")
	assert.Nil(t, gw)
	require.Error(t, err)
	assert.True(t, errors.Is(err, paykit.ErrUnknownGateway), "error must wrap ErrUnknownGateway")
	assert.Contains(t, err.Error(), "non_existent_provider")
}

func TestGet_FactoryErrorPropagation(t *testing.T) {
	name := "test_failing_factory"
	expectedErr := errors.New("initialization failed: missing credentials")

	paykit.Register(name, func() (paykit.Gateway, error) {
		return nil, expectedErr
	})

	gw, err := paykit.Get(name)
	assert.Nil(t, gw)
	require.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr))
}

func TestRegisteredGateways(t *testing.T) {
	names := paykit.RegisteredGateways()
	require.NotEmpty(t, names)

	// Verify sorting order
	for i := 1; i < len(names); i++ {
		assert.True(t, names[i-1] <= names[i], "registered names must be sorted alphabetically")
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, _ = paykit.Get("test_provider_success")
		}(i)

		go func() {
			defer wg.Done()
			_ = paykit.RegisteredGateways()
		}()
	}

	wg.Wait()
}
