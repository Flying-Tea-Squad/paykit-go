package mpesa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go"
)

var _ paykit.Gateway = (*Gateway)(nil)
var _ paykit.Disburser = (*Gateway)(nil)

func TestGateway_Charge_MapsSuccessfulSTKPushToPendingAction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mpesa/stkpush/v1/processrequest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"MerchantRequestID":"merchant-123",
			"CheckoutRequestID":"ws_CO_456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))
	}))
	defer server.Close()

	gw, err := New(Config{
		BaseURL:      server.URL,
		ShortCode:    "174379",
		Passkey:      "passkey-123",
		HTTPClient:   testHTTPClient(server.Client()),
		TokenManager: fakeTokenProvider{},
	})
	if err != nil {
		t.Fatalf("New returned unexpected error: %v", err)
	}

	resp, err := gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount:      paykit.NewKES(150.00),
		Phone:       "254700000001",
		Description: "Test STK push",
		Reference:   "INV-150",
		CallbackURL: "https://example.com/callback",
	})
	if err != nil {
		t.Fatalf("Charge returned unexpected error: %v", err)
	}

	if !resp.Success {
		t.Fatalf("expected successful charge initiation")
	}
	if resp.Status != paykit.StatusPendingAction {
		t.Fatalf("expected status %q, got %q", paykit.StatusPendingAction, resp.Status)
	}
	if resp.CheckoutRequestID != "ws_CO_456" {
		t.Fatalf("expected checkout_request_id %q, got %q", "ws_CO_456", resp.CheckoutRequestID)
	}
	if resp.TransactionID != "ws_CO_456" {
		t.Fatalf("expected transaction id %q, got %q", "ws_CO_456", resp.TransactionID)
	}
}

func TestGateway_New_RequiresPasskey(t *testing.T) {
	if _, err := New(Config{BaseURL: "https://example.com", ShortCode: "174379"}); err == nil {
		t.Fatal("expected New to reject empty passkey")
	}
}
