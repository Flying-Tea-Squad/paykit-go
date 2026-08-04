package mpesa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeTokenProvider struct{}

func TestSTKPushUsesCorrectendpoint(t *testing.T) {
	var (
		method string
		path   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path

		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`{
		"SellerRequestID": "123",
		"OutRequestID":"456",
		"Response":"0",
		"Description":"Success",
		"Message": "Accepted"
		}`))
	}))

	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		HTTPClient:   server.Client(),
		tokenManager: fakeTokenProvider{},
	}

	req := STKPushRequest{
		BusinessShortCode: "174379",
		Amount:            1,
		PartyA:            "254700000001",
		PartyB:            "174379",
		PhoneNumber:       "254700000001",
		CallBackURL:       "https://example.com/callback",
		AccountReference:  "INV001",
		TransactionDesc:   "Payment",
	}

	_, err := client.STKPush(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if method != http.MethodPost {
		t.Errorf("expected POST, got %s", method)
	}

	actPath := "/mpesa/stkpush/v1/processrequest"

	if path != actPath {
		t.Errorf("expected this path %q, got %q", actPath, path)
	}
}

func TestSTKPushParsesSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`{
			"MerchantRequestID":"123",
			"CheckoutRequestID":"456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		HTTPClient:   server.Client(),
		tokenManager: fakeTokenProvider{},
	}

	req := STKPushRequest{
		BusinessShortCode: "174379",
		Amount:            1,
		PartyA:            "254700000001",
		PartyB:            "174379",
		PhoneNumber:       "254700000001",
		CallBackURL:       "https://example.com/callback",
		AccountReference:  "INV001",
		TransactionDesc:   "Payment",
	}

	resp, err := client.STKPush(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.MerchantRequestID != "123" {
		t.Errorf("expected MerchantRequestID to be %q, got %q", "123", resp.MerchantRequestID)
	}

	if resp.CheckoutRequestID != "456" {
		t.Errorf("expected CheckoutRequestID   %q, got %q", "456", resp.CheckoutRequestID)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("expected ResponseCode %q, got %q", "0", resp.ResponseCode)
	}
	if resp.CustomerMessage != "Accepted" {
		t.Errorf("expected CustomerMessage%q, got %q", "Accepted", resp.CustomerMessage)
	}
}

func TestSTKPushIncludesIdempotencyHeader(t *testing.T) {
	var (
		authorization  string
		idempotencyKey string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		idempotencyKey = r.Header.Get("Idempotency-Key")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"MerchantRequestID":"123",
			"CheckoutRequestID":"456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))
	}))

	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		HTTPClient:   server.Client(),
		tokenManager: fakeTokenProvider{},
	}

	req := STKPushRequest{
		IdempotencyKey:    "stk-push-001",
		BusinessShortCode: "174379",
		Amount:            1,
		PartyA:            "254700000001",
		PartyB:            "174379",
		PhoneNumber:       "254700000001",
		CallBackURL:       "https://example.com/callback",
		AccountReference:  "INV001",
		TransactionDesc:   "Payment",
	}

	_, err := client.STKPush(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idempotencyKey != "stk-push-001" {
		t.Errorf("expected Idempotency-Key header %q, got %q", "stk-push-001", idempotencyKey)
	}

	if authorization != "Bearer test-token" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer test-token", authorization)
	}
}

func TestSTKPushGeneratesTimestampPassword(t *testing.T) {
	const passkey = "known-passkey"

	var received STKPushRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"MerchantRequestID":"123",
			"CheckoutRequestID":"456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))
	}))

	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		HTTPClient:   server.Client(),
		passkey:      passkey,
		tokenManager: fakeTokenProvider{},
	}

	req := STKPushRequest{
		BusinessShortCode: "174379",
		Amount:            1,
		PartyA:            "254700000001",
		PartyB:            "174379",
		PhoneNumber:       "254700000001",
		CallBackURL:       "https://example.com/callback",
		AccountReference:  "INV001",
		TransactionDesc:   "Payment",
	}

	_, err := client.STKPush(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsedTime, err := time.Parse("20060102150405", received.Timestamp)
	if err != nil {
		t.Fatalf("expected a valid timestamp, got %q: %v", received.Timestamp, err)
	}

	if parsedTime.IsZero() {
		t.Fatal("expected parsed timestamp to be non-zero")
	}

	wantPassword := base64.StdEncoding.EncodeToString([]byte(
		req.BusinessShortCode + passkey + received.Timestamp,
	))
	if received.Password != wantPassword {
		t.Errorf("expected password %q, got %q", wantPassword, received.Password)
	}
}

func TestSTKPushReturnsErrorForNon2xxResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "client error", statusCode: http.StatusBadRequest},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				HTTPClient:   server.Client(),
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.STKPush(context.Background(), STKPushRequest{})
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

func (f fakeTokenProvider) GetAccessToken(ctx context.Context) (string, error) {
	return "test-token", nil
}
