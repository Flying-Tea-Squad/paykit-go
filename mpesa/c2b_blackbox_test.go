package mpesa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/mpesa"
)

func testHTTPClient(c *http.Client) *paykit.HTTPClient {
	hc := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		IdempotencyPolicy: paykit.IdempotencyPolicyAutoGenerate,
	})
	hc.SetTransport(c.Transport)
	return hc
}

func TestC2BRegisterURL_Blackbox(t *testing.T) {
	t.Parallel()

	var (
		registerCalls  int32
		lastShortCode  string
		lastRespType   string
		lastConfirmURL string
		lastValidURL   string
		lastIdemKey    string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_, _ = w.Write([]byte(`{"access_token":"bb-token-c2b-reg","expires_in":"3599"}`))
		case "/mpesa/c2b/v1/registerurl":
			atomic.AddInt32(&registerCalls, 1)

			if auth := r.Header.Get("Authorization"); auth != "Bearer bb-token-c2b-reg" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"errorMessage":"Unauthorized"}`))
				return
			}

			lastIdemKey = r.Header.Get("Idempotency-Key")

			var reqBody struct {
				ShortCode       string `json:"ShortCode"`
				ResponseType    string `json:"ResponseType"`
				ConfirmationURL string `json:"ConfirmationURL"`
				ValidationURL   string `json:"ValidationURL"`
			}
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorMessage":"Bad Request"}`))
				return
			}

			lastShortCode = reqBody.ShortCode
			lastRespType = reqBody.ResponseType
			lastConfirmURL = reqBody.ConfirmationURL
			lastValidURL = reqBody.ValidationURL

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"OriginatorConversationID": "ORIG-REG-001",
				"ConversationID": "CONV-REG-001",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tm := mpesa.NewTokenManager(testHTTPClient(server.Client()), "key", "secret", server.URL)
	client := mpesa.NewMpesaClient(server.URL, testHTTPClient(server.Client()), "passkey", tm)
	ctx := context.Background()

	t.Run("successful registration with default response type", func(t *testing.T) {
		resp, err := client.C2BRegisterURL(ctx, mpesa.C2BRegisterRequest{
			IdempotencyKey:  "idem-reg-1",
			ShortCode:       "600123",
			ConfirmationURL: "https://example.com/confirm",
			ValidationURL:   "https://example.com/validate",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.OriginatorConversationID != "ORIG-REG-001" {
			t.Errorf("got OriginatorConversationID %q, want ORIG-REG-001", resp.OriginatorConversationID)
		}
		if resp.ConversationID != "CONV-REG-001" {
			t.Errorf("got ConversationID %q, want CONV-REG-001", resp.ConversationID)
		}
		if resp.ResponseCode != "0" {
			t.Errorf("got ResponseCode %q, want 0", resp.ResponseCode)
		}
		if resp.ResponseDescription != "Success" {
			t.Errorf("got ResponseDescription %q, want Success", resp.ResponseDescription)
		}

		if lastShortCode != "600123" {
			t.Errorf("got ShortCode %q, want 600123", lastShortCode)
		}
		if lastRespType != mpesa.ResponseTypeCompleted {
			t.Errorf("got ResponseType %q, want %q", lastRespType, mpesa.ResponseTypeCompleted)
		}
		if lastConfirmURL != "https://example.com/confirm" {
			t.Errorf("got ConfirmationURL %q, want https://example.com/confirm", lastConfirmURL)
		}
		if lastValidURL != "https://example.com/validate" {
			t.Errorf("got ValidationURL %q, want https://example.com/validate", lastValidURL)
		}
		if lastIdemKey != "idem-reg-1" {
			t.Errorf("got Idempotency-Key %q, want idem-reg-1", lastIdemKey)
		}
	})

	t.Run("explicit response type Cancelled", func(t *testing.T) {
		resp, err := client.C2BRegisterURL(ctx, mpesa.C2BRegisterRequest{
			ShortCode:       "600123",
			ResponseType:    mpesa.ResponseTypeCancelled,
			ConfirmationURL: "https://example.com/confirm",
			ValidationURL:   "https://example.com/validate",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if lastRespType != mpesa.ResponseTypeCancelled {
			t.Errorf("got ResponseType %q, want %q", lastRespType, mpesa.ResponseTypeCancelled)
		}
	})

	t.Run("client validation errors", func(t *testing.T) {
		tests := []struct {
			name    string
			req     mpesa.C2BRegisterRequest
			wantErr string
		}{
			{
				name: "missing short code",
				req: mpesa.C2BRegisterRequest{
					ConfirmationURL: "https://example.com/c",
					ValidationURL:   "https://example.com/v",
				},
				wantErr: "short code is required",
			},
			{
				name: "missing confirmation URL",
				req: mpesa.C2BRegisterRequest{
					ShortCode:     "600123",
					ValidationURL: "https://example.com/v",
				},
				wantErr: "confirmation URL is required",
			},
			{
				name: "missing validation URL",
				req: mpesa.C2BRegisterRequest{
					ShortCode:       "600123",
					ConfirmationURL: "https://example.com/c",
				},
				wantErr: "validation URL is required",
			},
			{
				name: "invalid response type",
				req: mpesa.C2BRegisterRequest{
					ShortCode:       "600123",
					ResponseType:    "InvalidType",
					ConfirmationURL: "https://example.com/c",
					ValidationURL:   "https://example.com/v",
				},
				wantErr: "invalid response type",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := client.C2BRegisterURL(ctx, tc.req)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestC2BSimulate_Blackbox(t *testing.T) {
	t.Parallel()

	var (
		simulateCalls int32
		lastShortCode string
		lastCommandID string
		lastAmount    int
		lastMsisdn    string
		lastBillRef   string
		lastIdemKey   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_, _ = w.Write([]byte(`{"access_token":"bb-token-c2b-sim","expires_in":"3599"}`))
		case "/mpesa/c2b/v1/simulate":
			atomic.AddInt32(&simulateCalls, 1)

			if auth := r.Header.Get("Authorization"); auth != "Bearer bb-token-c2b-sim" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"errorMessage":"Unauthorized"}`))
				return
			}

			lastIdemKey = r.Header.Get("Idempotency-Key")

			var reqBody struct {
				ShortCode     string `json:"ShortCode"`
				CommandID     string `json:"CommandID"`
				Amount        int    `json:"Amount"`
				Msisdn        string `json:"Msisdn"`
				BillRefNumber string `json:"BillRefNumber"`
			}
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorMessage":"Bad Request"}`))
				return
			}

			lastShortCode = reqBody.ShortCode
			lastCommandID = reqBody.CommandID
			lastAmount = reqBody.Amount
			lastMsisdn = reqBody.Msisdn
			lastBillRef = reqBody.BillRefNumber

			w.WriteHeader(http.StatusOK)
			// Returning Daraja's notorious typo "OriginatorCoversationID" to verify fallback in blackbox flow
			_, _ = w.Write([]byte(`{
				"OriginatorCoversationID": "ORIG-SIM-001",
				"ConversationID": "CONV-SIM-001",
				"ResponseCode": "0",
				"ResponseDescription": "Simulation successfully executed"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tm := mpesa.NewTokenManager(testHTTPClient(server.Client()), "key", "secret", server.URL)
	client := mpesa.NewMpesaClient(server.URL, testHTTPClient(server.Client()), "passkey", tm)
	ctx := context.Background()

	t.Run("successful simulation with default command ID and typo fallback", func(t *testing.T) {
		resp, err := client.C2BSimulate(ctx, mpesa.C2BSimulateRequest{
			IdempotencyKey: "idem-sim-1",
			ShortCode:      "600123",
			Amount:         250,
			Msisdn:         "254700000001",
			BillRefNumber:  "INV-1001",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.OriginatorConversationID != "ORIG-SIM-001" {
			t.Errorf("got OriginatorConversationID %q, want ORIG-SIM-001", resp.OriginatorConversationID)
		}
		if resp.ConversationID != "CONV-SIM-001" {
			t.Errorf("got ConversationID %q, want CONV-SIM-001", resp.ConversationID)
		}
		if resp.ResponseCode != "0" {
			t.Errorf("got ResponseCode %q, want 0", resp.ResponseCode)
		}
		if resp.ResponseDescription != "Simulation successfully executed" {
			t.Errorf("got ResponseDescription %q, want Simulation successfully executed", resp.ResponseDescription)
		}

		if lastShortCode != "600123" {
			t.Errorf("got ShortCode %q, want 600123", lastShortCode)
		}
		if lastCommandID != mpesa.CommandIDCustomerPayBillOnline {
			t.Errorf("got CommandID %q, want %q", lastCommandID, mpesa.CommandIDCustomerPayBillOnline)
		}
		if lastAmount != 250 {
			t.Errorf("got Amount %d, want 250", lastAmount)
		}
		if lastMsisdn != "254700000001" {
			t.Errorf("got Msisdn %q, want 254700000001", lastMsisdn)
		}
		if lastBillRef != "INV-1001" {
			t.Errorf("got BillRefNumber %q, want INV-1001", lastBillRef)
		}
		if lastIdemKey != "idem-sim-1" {
			t.Errorf("got Idempotency-Key %q, want idem-sim-1", lastIdemKey)
		}
	})

	t.Run("explicit command ID CustomerBuyGoodsOnline", func(t *testing.T) {
		resp, err := client.C2BSimulate(ctx, mpesa.C2BSimulateRequest{
			ShortCode: "600123",
			CommandID: mpesa.CommandIDCustomerBuyGoodsOnline,
			Amount:    500,
			Msisdn:    "254700000002",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if lastCommandID != mpesa.CommandIDCustomerBuyGoodsOnline {
			t.Errorf("got CommandID %q, want %q", lastCommandID, mpesa.CommandIDCustomerBuyGoodsOnline)
		}
	})

	t.Run("backward compatible CB2SimulateRequest alias", func(t *testing.T) {
		legacyReq := mpesa.CB2SimulateRequest{
			ShortCode: "600123",
			Amount:    100,
			Msisdn:    "254700000001",
		}
		resp, err := client.C2BSimulate(ctx, legacyReq)
		if err != nil {
			t.Fatalf("unexpected error with legacy type: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
	})

	t.Run("client validation errors", func(t *testing.T) {
		tests := []struct {
			name    string
			req     mpesa.C2BSimulateRequest
			wantErr string
		}{
			{
				name: "missing short code",
				req: mpesa.C2BSimulateRequest{
					Amount: 100,
					Msisdn: "254700000001",
				},
				wantErr: "short code is required",
			},
			{
				name: "missing MSISDN",
				req: mpesa.C2BSimulateRequest{
					ShortCode: "600123",
					Amount:    100,
				},
				wantErr: "MSISDN is required",
			},
			{
				name: "zero amount",
				req: mpesa.C2BSimulateRequest{
					ShortCode: "600123",
					Amount:    0,
					Msisdn:    "254700000001",
				},
				wantErr: "amount must be greater than zero",
			},
			{
				name: "negative amount",
				req: mpesa.C2BSimulateRequest{
					ShortCode: "600123",
					Amount:    -10,
					Msisdn:    "254700000001",
				},
				wantErr: "amount must be greater than zero",
			},
			{
				name: "invalid command ID",
				req: mpesa.C2BSimulateRequest{
					ShortCode: "600123",
					Amount:    100,
					Msisdn:    "254700000001",
					CommandID: "InvalidCommand",
				},
				wantErr: "invalid command ID",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := client.C2BSimulate(ctx, tc.req)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestC2B_HTTPErrorSurfacing_Blackbox(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_, _ = w.Write([]byte(`{"access_token":"bb-token-c2b-err","expires_in":"3599"}`))
		case "/mpesa/c2b/v1/registerurl":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errorCode":"400.002.02","errorMessage":"Invalid ValidationURL hostname"}`))
		case "/mpesa/c2b/v1/simulate":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errorCode":"403.000.01","errorMessage":"Sandbox simulation limit exceeded"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tm := mpesa.NewTokenManager(testHTTPClient(server.Client()), "key", "secret", server.URL)
	client := mpesa.NewMpesaClient(server.URL, testHTTPClient(server.Client()), "passkey", tm)
	ctx := context.Background()

	t.Run("C2BRegisterURL surfaces error response body", func(t *testing.T) {
		_, err := client.C2BRegisterURL(ctx, mpesa.C2BRegisterRequest{
			ShortCode:       "600123",
			ConfirmationURL: "https://example.com/c",
			ValidationURL:   "https://invalid.local/v",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Invalid ValidationURL hostname") {
			t.Errorf("expected error to contain API diagnostic body, got: %v", err)
		}
		if !strings.HasPrefix(err.Error(), "mpesa: unexpected status code 400:") {
			t.Errorf("expected error prefix 'mpesa: unexpected status code 400:', got: %v", err)
		}
	})

	t.Run("C2BSimulate surfaces error response body", func(t *testing.T) {
		_, err := client.C2BSimulate(ctx, mpesa.C2BSimulateRequest{
			ShortCode: "600123",
			Amount:    100,
			Msisdn:    "254700000001",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Sandbox simulation limit exceeded") {
			t.Errorf("expected error to contain API diagnostic body, got: %v", err)
		}
		if !strings.HasPrefix(err.Error(), "mpesa: unexpected status code 403:") {
			t.Errorf("expected error prefix 'mpesa: unexpected status code 403:', got: %v", err)
		}
	})
}
