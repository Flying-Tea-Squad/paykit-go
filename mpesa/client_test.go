package mpesa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeTokenProvider struct{}

// validSTKPushRequest returns a request that passes all STKPush validations.
func validSTKPushRequest() STKPushRequest {
	return STKPushRequest{
		BusinessShortCode: "174379",
		Amount:            1,
		PartyA:            "254700000001",
		PartyB:            "174379",
		PhoneNumber:       "254700000001",
		CallBackURL:       "https://example.com/callback",
		AccountReference:  "INV001",
		TransactionDesc:   "Payment",
	}
}

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
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "test-passkey",
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
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "test-passkey",
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
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "test-passkey",
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
		httpClient:   testHTTPClient(server.Client()),
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
				httpClient:   testHTTPClient(server.Client()),
				passkey:      "test-passkey",
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.STKPush(context.Background(), validSTKPushRequest())
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

func TestSTKPushValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*STKPushRequest)
		wantErr string
	}{
		{name: "missing business short code", mutate: func(r *STKPushRequest) { r.BusinessShortCode = "  " }, wantErr: "business short code is required"},
		{name: "zero amount", mutate: func(r *STKPushRequest) { r.Amount = 0 }, wantErr: "amount must be greater than zero"},
		{name: "negative amount", mutate: func(r *STKPushRequest) { r.Amount = -10 }, wantErr: "amount must be greater than zero"},
		{name: "missing party A", mutate: func(r *STKPushRequest) { r.PartyA = "" }, wantErr: "party A is required"},
		{name: "missing party B", mutate: func(r *STKPushRequest) { r.PartyB = "" }, wantErr: "party B is required"},
		{name: "missing phone number", mutate: func(r *STKPushRequest) { r.PhoneNumber = "" }, wantErr: "phone number is required"},
		{name: "invalid phone number", mutate: func(r *STKPushRequest) { r.PhoneNumber = "12345" }, wantErr: "not a valid Kenyan"},
		{name: "missing callback URL", mutate: func(r *STKPushRequest) { r.CallBackURL = " " }, wantErr: "callback URL is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requestCount, 1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := &Client{
				baseURL:      server.URL,
				httpClient:   testHTTPClient(server.Client()),
				passkey:      "passkey",
				tokenManager: fakeTokenProvider{},
			}

			req := validSTKPushRequest()
			tt.mutate(&req)

			resp, err := client.STKPush(context.Background(), req)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if resp != nil {
				t.Fatalf("expected nil response, got %#v", resp)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
			if got := atomic.LoadInt32(&requestCount); got != 0 {
				t.Errorf("expected no HTTP request for invalid input, got %d", got)
			}
		})
	}
}

func TestSTKPushTrimsTrailingSlashFromBaseURL(t *testing.T) {
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ResponseCode":"0"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL + "/",
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "passkey",
		tokenManager: fakeTokenProvider{},
	}

	if _, err := client.STKPush(context.Background(), validSTKPushRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/mpesa/stkpush/v1/processrequest"
	if path != want {
		t.Errorf("expected path %q, got %q", want, path)
	}
}

func TestSTKPushNormalizesWhitespaceAndSchemelessBaseURL(t *testing.T) {
	var path string

	// TLS server because a scheme-less base URL defaults to https.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ResponseCode":"0"}`))
	}))
	defer server.Close()

	schemeLess := strings.TrimPrefix(server.URL, "https://")
	client := &Client{
		baseURL:      "  " + schemeLess + "  ",
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "passkey",
		tokenManager: fakeTokenProvider{},
	}

	if _, err := client.STKPush(context.Background(), validSTKPushRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/mpesa/stkpush/v1/processrequest"
	if path != want {
		t.Errorf("expected path %q, got %q", want, path)
	}
}

func TestSTKPushRejectsInvalidBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr string
	}{
		{name: "empty", baseURL: "", wantErr: "base URL is required"},
		{name: "whitespace only", baseURL: "   ", wantErr: "base URL is required"},
		{name: "non-http scheme", baseURL: "ftp://example.com", wantErr: "not supported"},
		{name: "missing host", baseURL: "https://", wantErr: "missing a host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				baseURL:      tt.baseURL,
				httpClient:   testHTTPClient(http.DefaultClient),
				passkey:      "passkey",
				tokenManager: fakeTokenProvider{},
			}

			resp, err := client.STKPush(context.Background(), validSTKPushRequest())
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if resp != nil {
				t.Fatalf("expected nil response, got %#v", resp)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "plain https", in: "https://api.safaricom.co.ke", want: "https://api.safaricom.co.ke"},
		{name: "trailing slash", in: "https://api.safaricom.co.ke/", want: "https://api.safaricom.co.ke"},
		{name: "multiple trailing slashes", in: "https://api.safaricom.co.ke///", want: "https://api.safaricom.co.ke"},
		{name: "surrounding whitespace", in: "  https://api.safaricom.co.ke/  ", want: "https://api.safaricom.co.ke"},
		{name: "missing scheme defaults to https", in: "sandbox.safaricom.co.ke", want: "https://sandbox.safaricom.co.ke"},
		{name: "http scheme preserved", in: "http://localhost:8080/", want: "http://localhost:8080"},
		{name: "empty", in: "", wantErr: true},
		{name: "whitespace only", in: "\t ", wantErr: true},
		{name: "non-http scheme", in: "ftp://example.com", wantErr: true},
		{name: "missing host", in: "https://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeBaseURL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %q", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("normalizeBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSTKPushPreservesPresetPasswordAndTimestamp(t *testing.T) {
	var received STKPushRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ResponseCode":"0"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "some-passkey",
		tokenManager: fakeTokenProvider{},
	}

	req := validSTKPushRequest()
	req.Timestamp = "20240102030405"
	req.Password = "preset-password"

	if _, err := client.STKPush(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Timestamp != "20240102030405" {
		t.Errorf("expected preset timestamp to be preserved, got %q", received.Timestamp)
	}
	if received.Password != "preset-password" {
		t.Errorf("expected preset password to be preserved, got %q", received.Password)
	}
}

func TestSTKPushRequiresPasskeyWhenPasswordOmitted(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "",
		tokenManager: fakeTokenProvider{},
	}

	resp, err := client.STKPush(context.Background(), validSTKPushRequest())
	if err == nil {
		t.Fatal("expected error for missing passkey, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if !strings.Contains(err.Error(), "passkey is required") {
		t.Errorf("expected passkey error, got %q", err.Error())
	}
	if got := atomic.LoadInt32(&requestCount); got != 0 {
		t.Errorf("expected no HTTP request when passkey is missing, got %d", got)
	}
}

func TestSTKPushOversizedResponseIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return far more than the client-side response limit.
		_, _ = w.Write(make([]byte, 2<<20))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "passkey",
		tokenManager: fakeTokenProvider{},
	}

	resp, err := client.STKPush(context.Background(), validSTKPushRequest())
	if err == nil {
		t.Fatal("expected error for oversized response body, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected bounded-response error, got %q", err.Error())
	}
}

func TestNewMpesaClientDefaultsToOptionalIdempotencyPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ResponseCode":"0"}`))
	}))
	defer server.Close()

	// A nil httpClient must produce a client whose default idempotency policy
	// allows mutating requests without an Idempotency-Key (Optional, matching
	// mpesa.New in gateway.go) rather than rejecting them (Require).
	client := NewMpesaClient(server.URL, nil, "passkey", fakeTokenProvider{})

	req := validSTKPushRequest() // no IdempotencyKey set
	if _, err := client.STKPush(context.Background(), req); err != nil {
		t.Fatalf("expected STK push without idempotency key to succeed under the optional policy, got: %v", err)
	}
}
