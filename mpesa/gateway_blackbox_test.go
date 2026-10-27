package mpesa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/mpesa"
)

func TestMpesaGateway_CapabilityInterfaceConformance(t *testing.T) {
	// Verify compile-time and runtime interface assertions from consumer perspective
	var _ paykit.Gateway = (*mpesa.Gateway)(nil)
	var _ paykit.Disburser = (*mpesa.Gateway)(nil)
	var _ paykit.Gateway = (*mpesa.MpesaGateway)(nil)
	var _ paykit.Disburser = (*mpesa.MpesaGateway)(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_, _ = w.Write([]byte(`{"access_token":"token-123","expires_in":"3599"}`))
		case "/mpesa/stkpush/v1/processrequest":
			_, _ = w.Write([]byte(`{
				"MerchantRequestID": "m-123",
				"CheckoutRequestID": "ws_CO_123",
				"ResponseCode": "0",
				"ResponseDescription": "Success",
				"CustomerMessage": "Please check your phone"
			}`))
		case "/mpesa/b2c/v1/paymentrequest":
			_, _ = w.Write([]byte(`{
				"ConversationID": "AG_B2C_999",
				"OriginatorConversationID": "ORIG_B2C_999",
				"ResponseCode": "0",
				"ResponseDescription": "Accepted"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Direct typed constructor
	gw, err := mpesa.New(mpesa.Config{
		ConsumerKey:       "test-consumer-key",
		ConsumerSecret:    "test-consumer-secret",
		Passkey:           "test-passkey",
		BusinessShortCode: "174379",
		BaseURL:           server.URL,
		CallbackURL:       "https://example.com/stk-callback",
		InitiatorName:     "operator",
		SecurityCredential: "encrypted-password",
		HTTPClient: paykit.NewHTTPClient(paykit.HTTPClientConfig{
			IdempotencyPolicy: paykit.IdempotencyPolicyAutoGenerate,
		}),
	})
	if err != nil {
		t.Fatalf("failed to construct gateway: %v", err)
	}

	// 1. Core Gateway Name assertion
	if gw.Name() != "mpesa" {
		t.Errorf("expected gateway name mpesa, got %s", gw.Name())
	}

	// 2. Dynamic capability discovery via type assertion
	var genericGW paykit.Gateway = gw
	disburser, ok := genericGW.(paykit.Disburser)
	if !ok {
		t.Fatal("expected gateway to satisfy paykit.Disburser interface")
	}

	ctx := context.Background()

	// 3. Charge capability invocation
	chargeResp, err := genericGW.Charge(ctx, &paykit.ChargeRequest{
		Amount:      paykit.NewKES(100),
		Phone:       "+254712345678",
		Reference:   "INV-999",
		Description: "Blackbox test charge",
	})
	if err != nil {
		t.Fatalf("Charge failed: %v", err)
	}
	if !chargeResp.Success {
		t.Error("expected successful charge initiation")
	}
	if chargeResp.Status != paykit.StatusPendingAction {
		t.Errorf("expected StatusPendingAction, got %s", chargeResp.Status)
	}
	if chargeResp.CheckoutRequestID != "ws_CO_123" {
		t.Errorf("unexpected CheckoutRequestID: %s", chargeResp.CheckoutRequestID)
	}

	// 4. Disburse capability invocation
	disbResp, err := disburser.Disburse(ctx, &paykit.DisbursementRequest{
		Amount:      paykit.NewKES(50),
		Phone:       "0712345678",
		Description: "Blackbox test payout",
	})
	if err != nil {
		t.Fatalf("Disburse failed: %v", err)
	}
	if !disbResp.Success {
		t.Error("expected successful disbursement initiation")
	}
	if disbResp.Status != paykit.StatusProcessing {
		t.Errorf("expected StatusProcessing, got %s", disbResp.Status)
	}
	if disbResp.TransactionID != "AG_B2C_999" {
		t.Errorf("unexpected TransactionID: %s", disbResp.TransactionID)
	}
}
