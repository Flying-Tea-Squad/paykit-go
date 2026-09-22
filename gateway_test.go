package paykit_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Flying-Tea-Squad/paykit-go"
)

// mockFullGateway implements Gateway, Disburser, WebhookHandler, and BalanceChecker.
type mockFullGateway struct {
	name string
}

func (m *mockFullGateway) Name() string {
	return m.name
}

func (m *mockFullGateway) Charge(ctx context.Context, req *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, errors.New("invalid amount")
	}
	return &paykit.ChargeResponse{
		Success:       true,
		Message:       "Charge initiated",
		TransactionID: "txn_charge_123",
		Status:        "pending",
		Metadata: map[string]any{
			"phone": req.Phone,
		},
	}, nil
}

func (m *mockFullGateway) QueryStatus(ctx context.Context, req *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &paykit.StatusResponse{
		Success:       true,
		Message:       "Transaction completed",
		TransactionID: req.TransactionID,
		Status:        "completed",
	}, nil
}

func (m *mockFullGateway) Refund(ctx context.Context, req *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &paykit.RefundResponse{
		Success:       true,
		RefundID:      "ref_456",
		TransactionID: req.TransactionID,
		Status:        "completed",
	}, nil
}

func (m *mockFullGateway) Disburse(ctx context.Context, req *paykit.DisbursementRequest) (*paykit.DisbursementResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, errors.New("invalid disbursement amount")
	}
	return &paykit.DisbursementResponse{
		Success:       true,
		Message:       "Payout scheduled",
		TransactionID: "disb_789",
		Status:        "pending",
	}, nil
}

func (m *mockFullGateway) ParseAndVerify(r *http.Request) (*paykit.Event, error) {
	if r.Header.Get("X-Signature") != "valid_token" {
		return nil, errors.New("invalid webhook signature")
	}
	return &paykit.Event{
		ID:            "evt_001",
		Provider:      m.name,
		Type:          "charge.completed",
		TransactionID: "txn_charge_123",
		Amount:        1000,
		Currency:      "KES",
		Timestamp:     time.Now().UTC(),
	}, nil
}

func (m *mockFullGateway) CheckBalance(ctx context.Context, req *paykit.BalanceRequest) (*paykit.BalanceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &paykit.BalanceResponse{
		Success:   true,
		Currency:  "KES",
		Balance:   500000,
		Available: 450000,
	}, nil
}

// mockChargeOnlyGateway implements only Gateway (no Disburser, WebhookHandler, BalanceChecker).
type mockChargeOnlyGateway struct{}

func (m *mockChargeOnlyGateway) Name() string { return "charge_only" }
func (m *mockChargeOnlyGateway) Charge(ctx context.Context, req *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	return &paykit.ChargeResponse{Success: true, TransactionID: "co_1"}, nil
}
func (m *mockChargeOnlyGateway) QueryStatus(ctx context.Context, req *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	return &paykit.StatusResponse{Success: true}, nil
}
func (m *mockChargeOnlyGateway) Refund(ctx context.Context, req *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	return &paykit.RefundResponse{Success: true}, nil
}

// Compile-time interface satisfaction guarantees
var (
	_ paykit.Gateway        = (*mockFullGateway)(nil)
	_ paykit.Disburser      = (*mockFullGateway)(nil)
	_ paykit.WebhookHandler = (*mockFullGateway)(nil)
	_ paykit.BalanceChecker = (*mockFullGateway)(nil)

	_ paykit.Gateway = (*mockChargeOnlyGateway)(nil)
)

func TestGatewayCapabilities(t *testing.T) {
	t.Parallel()

	fullGw := &mockFullGateway{name: "full_provider"}
	chargeOnlyGw := &mockChargeOnlyGateway{}

	t.Run("Gateway Interface Methods", func(t *testing.T) {
		ctx := context.Background()

		// Test Charge
		chargeResp, err := fullGw.Charge(ctx, &paykit.ChargeRequest{
			Amount:   2500,
			Currency: "KES",
			Phone:    "+254712345678",
		})
		if err != nil {
			t.Fatalf("unexpected error charging: %v", err)
		}
		if !chargeResp.Success || chargeResp.TransactionID != "txn_charge_123" {
			t.Errorf("unexpected charge response: %+v", chargeResp)
		}

		// Test QueryStatus
		statusResp, err := fullGw.QueryStatus(ctx, &paykit.StatusRequest{
			TransactionID: "txn_charge_123",
		})
		if err != nil {
			t.Fatalf("unexpected error querying status: %v", err)
		}
		if statusResp.Status != "completed" {
			t.Errorf("unexpected status response: %+v", statusResp)
		}

		// Test Refund
		refundResp, err := fullGw.Refund(ctx, &paykit.RefundRequest{
			TransactionID: "txn_charge_123",
			Amount:        2500,
		})
		if err != nil {
			t.Fatalf("unexpected error refunding: %v", err)
		}
		if refundResp.RefundID != "ref_456" {
			t.Errorf("unexpected refund response: %+v", refundResp)
		}
	})

	t.Run("Disburser Capability Detection", func(t *testing.T) {
		ctx := context.Background()

		// Provider with Disburser
		if disburser, ok := any(fullGw).(paykit.Disburser); ok {
			payoutResp, err := disburser.Disburse(ctx, &paykit.DisbursementRequest{
				Amount:   5000,
				Currency: "KES",
				Phone:    "+254798765432",
			})
			if err != nil {
				t.Fatalf("unexpected error disbursing: %v", err)
			}
			if payoutResp.TransactionID != "disb_789" {
				t.Errorf("unexpected payout response: %+v", payoutResp)
			}
		} else {
			t.Fatal("expected fullGw to satisfy paykit.Disburser")
		}

		// Charge-only provider without Disburser
		if _, ok := any(chargeOnlyGw).(paykit.Disburser); ok {
			t.Fatal("expected chargeOnlyGw to NOT satisfy paykit.Disburser")
		}
	})

	t.Run("WebhookHandler Capability Detection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
		req.Header.Set("X-Signature", "valid_token")

		handler, ok := any(fullGw).(paykit.WebhookHandler)
		if !ok {
			t.Fatal("expected fullGw to satisfy paykit.WebhookHandler")
		}

		evt, err := handler.ParseAndVerify(req)
		if err != nil {
			t.Fatalf("unexpected error parsing webhook: %v", err)
		}
		if evt.Type != "charge.completed" || evt.Amount != 1000 {
			t.Errorf("unexpected event: %+v", evt)
		}

		// Invalid signature check
		badReq := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
		_, err = handler.ParseAndVerify(badReq)
		if err == nil {
			t.Fatal("expected error with bad webhook signature")
		}

		// Charge-only check
		if _, ok := any(chargeOnlyGw).(paykit.WebhookHandler); ok {
			t.Fatal("expected chargeOnlyGw to NOT satisfy paykit.WebhookHandler")
		}
	})

	t.Run("BalanceChecker Capability Detection", func(t *testing.T) {
		ctx := context.Background()

		checker, ok := any(fullGw).(paykit.BalanceChecker)
		if !ok {
			t.Fatal("expected fullGw to satisfy paykit.BalanceChecker")
		}

		balResp, err := checker.CheckBalance(ctx, &paykit.BalanceRequest{Currency: "KES"})
		if err != nil {
			t.Fatalf("unexpected error checking balance: %v", err)
		}
		if balResp.Balance != 500000 || balResp.Available != 450000 {
			t.Errorf("unexpected balance response: %+v", balResp)
		}

		if _, ok := any(chargeOnlyGw).(paykit.BalanceChecker); ok {
			t.Fatal("expected chargeOnlyGw to NOT satisfy paykit.BalanceChecker")
		}
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := fullGw.Charge(ctx, &paykit.ChargeRequest{Amount: 100})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got: %v", err)
		}

		var gw paykit.Gateway = fullGw
		disburser := gw.(paykit.Disburser)
		_, err = disburser.Disburse(ctx, &paykit.DisbursementRequest{Amount: 100})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got: %v", err)
		}
	})

	t.Run("JSON Serialization", func(t *testing.T) {
		req := paykit.ChargeRequest{
			Amount:         10000,
			Currency:       "KES",
			Phone:          "+254712345678",
			Description:    "Test order",
			Reference:      "ORD-101",
			IdempotencyKey: "idem-abc",
			CallbackURL:    "https://example.com/callback",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal ChargeRequest: %v", err)
		}

		var decoded paykit.ChargeRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal ChargeRequest: %v", err)
		}

		if decoded.Amount != req.Amount || decoded.Phone != req.Phone || decoded.Reference != req.Reference {
			t.Errorf("mismatched decoded charge request: %+v", decoded)
		}
	})
}
