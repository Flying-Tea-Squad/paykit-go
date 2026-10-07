package gateway_test

import (
	"context"
	"fmt"
	"testing"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyGateway struct {
	name string
}

func (d *dummyGateway) Name() string { return d.name }
func (d *dummyGateway) Charge(_ context.Context, _ *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	return nil, nil
}
func (d *dummyGateway) QueryStatus(_ context.Context, _ *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	return nil, nil
}
func (d *dummyGateway) Refund(_ context.Context, _ *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	return nil, nil
}

func TestLegacyGatewayRegistry(t *testing.T) {
	t.Run("register and instantiate success", func(t *testing.T) {
		name := "legacy_dummy"
		gateway.Register(name, func(config any) (paykit.Gateway, error) {
			return &dummyGateway{name: name}, nil
		})

		gw, err := gateway.New(name, nil)
		require.NoError(t, err)
		assert.Equal(t, name, gw.Name())
	})

	t.Run("unknown provider returns error", func(t *testing.T) {
		gw, err := gateway.New("unknown_legacy_provider", nil)
		assert.Nil(t, gw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})

	t.Run("invalid registration panics", func(t *testing.T) {
		assert.Panics(t, func() {
			gateway.Register("", nil)
		})
		assert.Panics(t, func() {
			gateway.Register("valid_name", nil)
		})

		dupName := "duplicate_legacy"
		gateway.Register(dupName, func(config any) (paykit.Gateway, error) {
			return &dummyGateway{name: dupName}, nil
		})
		assert.PanicsWithValue(t, fmt.Sprintf("gateway: provider %q already registered", dupName), func() {
			gateway.Register(dupName, func(config any) (paykit.Gateway, error) {
				return &dummyGateway{name: dupName}, nil
			})
		})
	})
}
