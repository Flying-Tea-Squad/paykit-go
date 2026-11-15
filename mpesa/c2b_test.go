package mpesa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type errorTokenProvider struct {
	err error
}

func (e errorTokenProvider) GetAccessToken(ctx context.Context) (string, error) {
	return "", e.err
}

func TestC2BRegisterURL_Success(t *testing.T) {
	tests := []struct {
		name                 string
		req                  C2BRegisterRequest
		serverResponse       string
		wantShortCode        string
		wantResponseType     string
		wantConfirmationURL  string
		wantValidationURL    string
		wantIdempotencyKey   string
		wantOriginatorConvID string
		wantConversationID   string
		wantResponseCode     string
		wantResponseDesc     string
	}{
		{
			name: "standard response with default ResponseType Completed",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "https://example.com/validation",
			},
			serverResponse: `{
				"OriginatorConversationID": "12345-67890-1",
				"ConversationID": "AG_2023_001",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantShortCode:        "600000",
			wantResponseType:     "Completed",
			wantConfirmationURL:  "https://example.com/confirmation",
			wantValidationURL:    "https://example.com/validation",
			wantOriginatorConvID: "12345-67890-1",
			wantConversationID:   "AG_2023_001",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "Daraja typo OriginatorCoversationID with explicit Cancelled and IdempotencyKey",
			req: C2BRegisterRequest{
				IdempotencyKey:  "idem-c2b-reg-001",
				ShortCode:       "600001",
				ResponseType:    "Cancelled",
				ConfirmationURL: "https://example.com/confirmation2",
				ValidationURL:   "https://example.com/validation2",
			},
			serverResponse: `{
				"OriginatorCoversationID": "typo-reg-002",
				"ConversationID": "AG_2023_002",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantShortCode:        "600001",
			wantResponseType:     "Cancelled",
			wantConfirmationURL:  "https://example.com/confirmation2",
			wantValidationURL:    "https://example.com/validation2",
			wantIdempotencyKey:   "idem-c2b-reg-001",
			wantOriginatorConvID: "typo-reg-002",
			wantConversationID:   "AG_2023_002",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "response without optional ResponseCode",
			req: C2BRegisterRequest{
				ShortCode:       "600002",
				ResponseType:    "Completed",
				ConfirmationURL: "https://example.com/confirmation3",
				ValidationURL:   "https://example.com/validation3",
			},
			serverResponse: `{
				"OriginatorCoversationID": "typo-reg-003",
				"ResponseDescription": "Registration accepted"
			}`,
			wantShortCode:        "600002",
			wantResponseType:     "Completed",
			wantConfirmationURL:  "https://example.com/confirmation3",
			wantValidationURL:    "https://example.com/validation3",
			wantOriginatorConvID: "typo-reg-003",
			wantConversationID:   "",
			wantResponseCode:     "",
			wantResponseDesc:     "Registration accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				gotMethod         string
				gotPath           string
				gotAuthHeader     string
				gotIdempotency    string
				gotContentType    string
				gotBody           C2BRegisterRequest
				serverCalledCount int32
			)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&serverCalledCount, 1)
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAuthHeader = r.Header.Get("Authorization")
				gotIdempotency = r.Header.Get("Idempotency-Key")
				gotContentType = r.Header.Get("Content-Type")

				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.C2BRegisterURL(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("unexpected error from C2BRegisterURL: %v", err)
			}

			if serverCalledCount != 1 {
				t.Fatalf("expected server to be called 1 time, got %d", serverCalledCount)
			}
			if gotMethod != http.MethodPost {
				t.Errorf("expected POST method, got %s", gotMethod)
			}
			if gotPath != "/mpesa/c2b/v1/registerurl" {
				t.Errorf("expected path /mpesa/c2b/v1/registerurl, got %s", gotPath)
			}
			if gotAuthHeader != "Bearer test-token" {
				t.Errorf("expected Authorization header 'Bearer test-token', got %q", gotAuthHeader)
			}
			if gotContentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", gotContentType)
			}
			if tt.wantIdempotencyKey != "" && gotIdempotency != tt.wantIdempotencyKey {
				t.Errorf("expected Idempotency-Key header %q, got %q", tt.wantIdempotencyKey, gotIdempotency)
			}

			// Validate request payload received by server
			if gotBody.ShortCode != tt.wantShortCode {
				t.Errorf("server received ShortCode %q, want %q", gotBody.ShortCode, tt.wantShortCode)
			}
			if gotBody.ResponseType != tt.wantResponseType {
				t.Errorf("server received ResponseType %q, want %q", gotBody.ResponseType, tt.wantResponseType)
			}
			if gotBody.ConfirmationURL != tt.wantConfirmationURL {
				t.Errorf("server received ConfirmationURL %q, want %q", gotBody.ConfirmationURL, tt.wantConfirmationURL)
			}
			if gotBody.ValidationURL != tt.wantValidationURL {
				t.Errorf("server received ValidationURL %q, want %q", gotBody.ValidationURL, tt.wantValidationURL)
			}

			// Validate decoded response struct
			if resp.OriginatorConversationID != tt.wantOriginatorConvID {
				t.Errorf("expected OriginatorConversationID %q, got %q", tt.wantOriginatorConvID, resp.OriginatorConversationID)
			}
			if resp.ConversationID != tt.wantConversationID {
				t.Errorf("expected ConversationID %q, got %q", tt.wantConversationID, resp.ConversationID)
			}
			if resp.ResponseCode != tt.wantResponseCode {
				t.Errorf("expected ResponseCode %q, got %q", tt.wantResponseCode, resp.ResponseCode)
			}
			if resp.ResponseDescription != tt.wantResponseDesc {
				t.Errorf("expected ResponseDescription %q, got %q", tt.wantResponseDesc, resp.ResponseDescription)
			}
		})
	}
}

func TestC2BRegisterURL_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     C2BRegisterRequest
		wantErr string
	}{
		{
			name: "empty ShortCode",
			req: C2BRegisterRequest{
				ShortCode:       "",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "https://example.com/validation",
			},
			wantErr: "short code is required",
		},
		{
			name: "whitespace-only ShortCode",
			req: C2BRegisterRequest{
				ShortCode:       "   ",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "https://example.com/validation",
			},
			wantErr: "short code is required",
		},
		{
			name: "empty ConfirmationURL",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ConfirmationURL: "",
				ValidationURL:   "https://example.com/validation",
			},
			wantErr: "confirmation URL is required",
		},
		{
			name: "whitespace-only ConfirmationURL",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ConfirmationURL: "\t  \n ",
				ValidationURL:   "https://example.com/validation",
			},
			wantErr: "confirmation URL is required",
		},
		{
			name: "empty ValidationURL",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "",
			},
			wantErr: "validation URL is required",
		},
		{
			name: "whitespace-only ValidationURL",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "   ",
			},
			wantErr: "validation URL is required",
		},
		{
			name: "invalid ResponseType Pending",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ResponseType:    "Pending",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "https://example.com/validation",
			},
			wantErr: "invalid response type",
		},
		{
			name: "invalid lowercase completed ResponseType",
			req: C2BRegisterRequest{
				ShortCode:       "600000",
				ResponseType:    "completed",
				ConfirmationURL: "https://example.com/confirmation",
				ValidationURL:   "https://example.com/validation",
			},
			wantErr: "invalid response type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverCalled int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&serverCalled, 1)
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.C2BRegisterURL(context.Background(), tt.req)
			if resp != nil {
				t.Fatalf("expected nil response on validation failure, got %#v", resp)
			}
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
			if atomic.LoadInt32(&serverCalled) != 0 {
				t.Errorf("server was called %d times, expected 0 calls on client-side validation failure", serverCalled)
			}
		})
	}
}

func TestC2BRegisterURL_Non2xxResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: http.StatusBadRequest},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
		{name: "forbidden", statusCode: http.StatusForbidden},
		{name: "not found", statusCode: http.StatusNotFound},
		{name: "internal server error", statusCode: http.StatusInternalServerError},
		{name: "service unavailable", statusCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				tokenManager: fakeTokenProvider{},
			}

			req := C2BRegisterRequest{
				ShortCode:       "600000",
				ConfirmationURL: "https://example.com/confirm",
				ValidationURL:   "https://example.com/validate",
			}

			resp, err := client.C2BRegisterURL(context.Background(), req)
			if resp != nil {
				t.Fatalf("expected nil response, got %#v", resp)
			}

			wantErr := fmt.Sprintf("unexpected status code: %d", tt.statusCode)
			if err == nil || err.Error() != wantErr {
				t.Fatalf("expected error %q, got %v", wantErr, err)
			}
		})
	}
}

func TestC2BSimulate_Success(t *testing.T) {
	tests := []struct {
		name                 string
		req                  C2BSimulateRequest
		serverResponse       string
		wantShortCode        string
		wantCommandID        string
		wantAmount           int
		wantMsisdn           string
		wantBillRefNumber    string
		wantIdempotencyKey   string
		wantOriginatorConvID string
		wantConversationID   string
		wantResponseCode     string
		wantResponseDesc     string
	}{
		{
			name: "standard response with default CommandID CustomerPayBillOnline",
			req: C2BSimulateRequest{
				ShortCode:     "600000",
				Amount:        100,
				Msisdn:        "254700000001",
				BillRefNumber: "INV-1001",
			},
			serverResponse: `{
				"OriginatorConversationID": "sim-conv-001",
				"ConversationID": "AG_SIM_001",
				"ResponseCode": "0",
				"ResponseDescription": "Accept the service request successfully."
			}`,
			wantShortCode:        "600000",
			wantCommandID:        "CustomerPayBillOnline",
			wantAmount:           100,
			wantMsisdn:           "254700000001",
			wantBillRefNumber:    "INV-1001",
			wantOriginatorConvID: "sim-conv-001",
			wantConversationID:   "AG_SIM_001",
			wantResponseCode:     "0",
			wantResponseDesc:     "Accept the service request successfully.",
		},
		{
			name: "Daraja typo OriginatorCoversationID with explicit BuyGoods and IdempotencyKey",
			req: C2BSimulateRequest{
				IdempotencyKey: "sim-idem-002",
				ShortCode:      "600002",
				CommandID:      "CustomerBuyGoodsOnline",
				Amount:         250,
				Msisdn:         "254700000002",
			},
			serverResponse: `{
				"OriginatorCoversationID": "typo-sim-conv-002",
				"ConversationID": "AG_SIM_002",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantShortCode:        "600002",
			wantCommandID:        "CustomerBuyGoodsOnline",
			wantAmount:           250,
			wantMsisdn:           "254700000002",
			wantIdempotencyKey:   "sim-idem-002",
			wantOriginatorConvID: "typo-sim-conv-002",
			wantConversationID:   "AG_SIM_002",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "response without optional ResponseCode",
			req: C2BSimulateRequest{
				ShortCode:     "600003",
				Amount:        50,
				Msisdn:        "254700000003",
				BillRefNumber: "ACC-50",
			},
			serverResponse: `{
				"OriginatorConversationID": "sim-conv-003",
				"ResponseDescription": "Simulated successfully"
			}`,
			wantShortCode:        "600003",
			wantCommandID:        "CustomerPayBillOnline",
			wantAmount:           50,
			wantMsisdn:           "254700000003",
			wantBillRefNumber:    "ACC-50",
			wantOriginatorConvID: "sim-conv-003",
			wantConversationID:   "",
			wantResponseCode:     "",
			wantResponseDesc:     "Simulated successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				gotMethod         string
				gotPath           string
				gotAuthHeader     string
				gotIdempotency    string
				gotContentType    string
				gotBody           C2BSimulateRequest
				serverCalledCount int32
			)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&serverCalledCount, 1)
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotAuthHeader = r.Header.Get("Authorization")
				gotIdempotency = r.Header.Get("Idempotency-Key")
				gotContentType = r.Header.Get("Content-Type")

				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.C2BSimulate(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("unexpected error from C2BSimulate: %v", err)
			}

			if serverCalledCount != 1 {
				t.Fatalf("expected server to be called 1 time, got %d", serverCalledCount)
			}
			if gotMethod != http.MethodPost {
				t.Errorf("expected POST method, got %s", gotMethod)
			}
			if gotPath != "/mpesa/c2b/v1/simulate" {
				t.Errorf("expected path /mpesa/c2b/v1/simulate, got %s", gotPath)
			}
			if gotAuthHeader != "Bearer test-token" {
				t.Errorf("expected Authorization header 'Bearer test-token', got %q", gotAuthHeader)
			}
			if gotContentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", gotContentType)
			}
			if tt.wantIdempotencyKey != "" && gotIdempotency != tt.wantIdempotencyKey {
				t.Errorf("expected Idempotency-Key header %q, got %q", tt.wantIdempotencyKey, gotIdempotency)
			}

			// Validate payload fields received by server
			if gotBody.ShortCode != tt.wantShortCode {
				t.Errorf("server received ShortCode %q, want %q", gotBody.ShortCode, tt.wantShortCode)
			}
			if gotBody.CommandID != tt.wantCommandID {
				t.Errorf("server received CommandID %q, want %q", gotBody.CommandID, tt.wantCommandID)
			}
			if gotBody.Amount != tt.wantAmount {
				t.Errorf("server received Amount %d, want %d", gotBody.Amount, tt.wantAmount)
			}
			if gotBody.Msisdn != tt.wantMsisdn {
				t.Errorf("server received Msisdn %q, want %q", gotBody.Msisdn, tt.wantMsisdn)
			}
			if tt.wantBillRefNumber != "" && gotBody.BillRefNumber != tt.wantBillRefNumber {
				t.Errorf("server received BillRefNumber %q, want %q", gotBody.BillRefNumber, tt.wantBillRefNumber)
			}

			// Validate decoded response struct
			if resp.OriginatorConversationID != tt.wantOriginatorConvID {
				t.Errorf("expected OriginatorConversationID %q, got %q", tt.wantOriginatorConvID, resp.OriginatorConversationID)
			}
			if resp.ConversationID != tt.wantConversationID {
				t.Errorf("expected ConversationID %q, got %q", tt.wantConversationID, resp.ConversationID)
			}
			if resp.ResponseCode != tt.wantResponseCode {
				t.Errorf("expected ResponseCode %q, got %q", tt.wantResponseCode, resp.ResponseCode)
			}
			if resp.ResponseDescription != tt.wantResponseDesc {
				t.Errorf("expected ResponseDescription %q, got %q", tt.wantResponseDesc, resp.ResponseDescription)
			}
		})
	}
}

func TestC2BSimulate_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     C2BSimulateRequest
		wantErr string
	}{
		{
			name: "empty ShortCode",
			req: C2BSimulateRequest{
				ShortCode: "",
				Amount:    100,
				Msisdn:    "254700000001",
			},
			wantErr: "short code is required",
		},
		{
			name: "whitespace-only ShortCode",
			req: C2BSimulateRequest{
				ShortCode: "   ",
				Amount:    100,
				Msisdn:    "254700000001",
			},
			wantErr: "short code is required",
		},
		{
			name: "empty Msisdn",
			req: C2BSimulateRequest{
				ShortCode: "600000",
				Amount:    100,
				Msisdn:    "",
			},
			wantErr: "MSISDN is required",
		},
		{
			name: "whitespace-only Msisdn",
			req: C2BSimulateRequest{
				ShortCode: "600000",
				Amount:    100,
				Msisdn:    " \t ",
			},
			wantErr: "MSISDN is required",
		},
		{
			name: "zero Amount",
			req: C2BSimulateRequest{
				ShortCode: "600000",
				Amount:    0,
				Msisdn:    "254700000001",
			},
			wantErr: "amount must be greater than zero",
		},
		{
			name: "negative Amount",
			req: C2BSimulateRequest{
				ShortCode: "600000",
				Amount:    -50,
				Msisdn:    "254700000001",
			},
			wantErr: "amount must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverCalled int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&serverCalled, 1)
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.C2BSimulate(context.Background(), tt.req)
			if resp != nil {
				t.Fatalf("expected nil response on validation failure, got %#v", resp)
			}
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
			if atomic.LoadInt32(&serverCalled) != 0 {
				t.Errorf("server was called %d times, expected 0 calls on client-side validation failure", serverCalled)
			}
		})
	}
}

func TestC2BSimulate_Non2xxResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: http.StatusBadRequest},
		{name: "unauthorized", statusCode: http.StatusUnauthorized},
		{name: "forbidden", statusCode: http.StatusForbidden},
		{name: "internal server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				tokenManager: fakeTokenProvider{},
			}

			req := C2BSimulateRequest{
				ShortCode: "600000",
				Amount:    100,
				Msisdn:    "254700000001",
			}

			resp, err := client.C2BSimulate(context.Background(), req)
			if resp != nil {
				t.Fatalf("expected nil response, got %#v", resp)
			}

			wantErr := fmt.Sprintf("unexpected status code: %d", tt.statusCode)
			if err == nil || err.Error() != wantErr {
				t.Fatalf("expected error %q, got %v", wantErr, err)
			}
		})
	}
}

func TestC2BResponses_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name                 string
		jsonData             string
		wantOriginatorConvID string
		wantConversationID   string
		wantResponseCode     string
		wantResponseDesc     string
		wantErr              bool
	}{
		{
			name: "standard OriginatorConversationID",
			jsonData: `{
				"OriginatorConversationID": "standard-conv-123",
				"ConversationID": "conv-456",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantOriginatorConvID: "standard-conv-123",
			wantConversationID:   "conv-456",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "Daraja typo OriginatorCoversationID",
			jsonData: `{
				"OriginatorCoversationID": "typo-conv-123",
				"ConversationID": "conv-789",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantOriginatorConvID: "typo-conv-123",
			wantConversationID:   "conv-789",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "both standard and typo present prefers standard",
			jsonData: `{
				"OriginatorConversationID": "standard-preferred",
				"OriginatorCoversationID": "typo-secondary",
				"ConversationID": "conv-both",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantOriginatorConvID: "standard-preferred",
			wantConversationID:   "conv-both",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "empty standard key falls back to typo key",
			jsonData: `{
				"OriginatorConversationID": "",
				"OriginatorCoversationID": "typo-fallback",
				"ConversationID": "conv-fallback",
				"ResponseCode": "0",
				"ResponseDescription": "Success"
			}`,
			wantOriginatorConvID: "typo-fallback",
			wantConversationID:   "conv-fallback",
			wantResponseCode:     "0",
			wantResponseDesc:     "Success",
		},
		{
			name: "optional ResponseCode omitted",
			jsonData: `{
				"OriginatorConversationID": "standard-no-code",
				"ConversationID": "conv-no-code",
				"ResponseDescription": "Accepted"
			}`,
			wantOriginatorConvID: "standard-no-code",
			wantConversationID:   "conv-no-code",
			wantResponseCode:     "",
			wantResponseDesc:     "Accepted",
		},
		{
			name:     "malformed JSON",
			jsonData: `{"OriginatorConversationID": `,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run("C2BRegisterResponse/"+tt.name, func(t *testing.T) {
			var resp C2BRegisterResponse
			err := json.Unmarshal([]byte(tt.jsonData), &resp)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if resp.OriginatorConversationID != tt.wantOriginatorConvID {
				t.Errorf("OriginatorConversationID = %q, want %q", resp.OriginatorConversationID, tt.wantOriginatorConvID)
			}
			if resp.ConversationID != tt.wantConversationID {
				t.Errorf("ConversationID = %q, want %q", resp.ConversationID, tt.wantConversationID)
			}
			if resp.ResponseCode != tt.wantResponseCode {
				t.Errorf("ResponseCode = %q, want %q", resp.ResponseCode, tt.wantResponseCode)
			}
			if resp.ResponseDescription != tt.wantResponseDesc {
				t.Errorf("ResponseDescription = %q, want %q", resp.ResponseDescription, tt.wantResponseDesc)
			}
		})

		t.Run("C2BSimulateResponse/"+tt.name, func(t *testing.T) {
			var resp C2BSimulateResponse
			err := json.Unmarshal([]byte(tt.jsonData), &resp)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if resp.OriginatorConversationID != tt.wantOriginatorConvID {
				t.Errorf("OriginatorConversationID = %q, want %q", resp.OriginatorConversationID, tt.wantOriginatorConvID)
			}
			if resp.ConversationID != tt.wantConversationID {
				t.Errorf("ConversationID = %q, want %q", resp.ConversationID, tt.wantConversationID)
			}
			if resp.ResponseCode != tt.wantResponseCode {
				t.Errorf("ResponseCode = %q, want %q", resp.ResponseCode, tt.wantResponseCode)
			}
			if resp.ResponseDescription != tt.wantResponseDesc {
				t.Errorf("ResponseDescription = %q, want %q", resp.ResponseDescription, tt.wantResponseDesc)
			}
		})
	}

	t.Run("nil receiver error", func(t *testing.T) {
		var regResp *C2BRegisterResponse
		if err := regResp.UnmarshalJSON([]byte(`{}`)); err == nil {
			t.Error("expected error for nil C2BRegisterResponse receiver, got nil")
		}

		var simResp *C2BSimulateResponse
		if err := simResp.UnmarshalJSON([]byte(`{}`)); err == nil {
			t.Error("expected error for nil C2BSimulateResponse receiver, got nil")
		}
	})
}

func TestCB2SimulateRequest_BackwardCompatibilityAlias(t *testing.T) {
	// Ensure CB2SimulateRequest can be used as C2BSimulateRequest without conversion.
	legacyReq := CB2SimulateRequest{
		IdempotencyKey: "compat-idem-key",
		ShortCode:      "600000",
		CommandID:      "CustomerPayBillOnline",
		Amount:         500,
		Msisdn:         "254700000001",
		BillRefNumber:  "ACC-COMPAT",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"OriginatorConversationID": "compat-conv-id",
			"ConversationID": "AG_COMPAT",
			"ResponseCode": "0",
			"ResponseDescription": "Success"
		}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		tokenManager: fakeTokenProvider{},
	}

	// Direct call to C2BSimulate with the aliased CB2SimulateRequest type
	resp, err := client.C2BSimulate(context.Background(), legacyReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.OriginatorConversationID != "compat-conv-id" {
		t.Errorf("expected OriginatorConversationID 'compat-conv-id', got %q", resp.OriginatorConversationID)
	}
}

func TestC2B_MissingTokenManager(t *testing.T) {
	client := &Client{
		baseURL:      "http://example.com",
		httpClient:   testHTTPClient(http.DefaultClient),
		tokenManager: nil,
	}

	_, err := client.C2BRegisterURL(context.Background(), C2BRegisterRequest{
		ShortCode:       "600000",
		ConfirmationURL: "https://example.com/c",
		ValidationURL:   "https://example.com/v",
	})
	if err == nil || !strings.Contains(err.Error(), "token manager is required") {
		t.Errorf("expected token manager error, got %v", err)
	}

	_, err = client.C2BSimulate(context.Background(), C2BSimulateRequest{
		ShortCode: "600000",
		Amount:    100,
		Msisdn:    "254700000001",
	})
	if err == nil || !strings.Contains(err.Error(), "token manager is required") {
		t.Errorf("expected token manager error, got %v", err)
	}
}

func TestC2B_TokenProviderError(t *testing.T) {
	authErr := errors.New("auth failure: invalid credentials")
	client := &Client{
		baseURL:      "http://example.com",
		httpClient:   testHTTPClient(http.DefaultClient),
		tokenManager: errorTokenProvider{err: authErr},
	}

	_, err := client.C2BRegisterURL(context.Background(), C2BRegisterRequest{
		ShortCode:       "600000",
		ConfirmationURL: "https://example.com/c",
		ValidationURL:   "https://example.com/v",
	})
	if !errors.Is(err, authErr) {
		t.Errorf("expected auth error %v, got %v", authErr, err)
	}

	_, err = client.C2BSimulate(context.Background(), C2BSimulateRequest{
		ShortCode: "600000",
		Amount:    100,
		Msisdn:    "254700000001",
	})
	if !errors.Is(err, authErr) {
		t.Errorf("expected auth error %v, got %v", authErr, err)
	}
}

func TestC2B_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		tokenManager: fakeTokenProvider{},
	}

	_, err := client.C2BRegisterURL(context.Background(), C2BRegisterRequest{
		ShortCode:       "600000",
		ConfirmationURL: "https://example.com/c",
		ValidationURL:   "https://example.com/v",
	})
	if err == nil {
		t.Error("expected JSON decode error for C2BRegisterURL, got nil")
	}

	_, err = client.C2BSimulate(context.Background(), C2BSimulateRequest{
		ShortCode: "600000",
		Amount:    100,
		Msisdn:    "254700000001",
	})
	if err == nil {
		t.Error("expected JSON decode error for C2BSimulate, got nil")
	}
}
