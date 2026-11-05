package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSTKPushQuery_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/mpesa/stkpushquery/v1/query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		var req STKPushQueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.CheckoutRequestID != "ws_CO_12345" {
			t.Errorf("unexpected CheckoutRequestID: %s", req.CheckoutRequestID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"ResponseCode": "0",
			"ResponseDescription": "The service request has been accepted successfully",
			"MerchantRequestID": "22205-5698992-2",
			"CheckoutRequestID": "ws_CO_12345",
			"ResultCode": "0",
			"ResultDesc": "The service request is processed successfully."
		}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "test-passkey",
		tokenManager: fakeTokenProvider{},
	}

	resp, err := client.STKPushQuery(context.Background(), STKPushQueryRequest{
		BusinessShortCode: "174379",
		CheckoutRequestID: "ws_CO_12345",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ResultCode != "0" {
		t.Errorf("expected ResultCode 0, got %s", resp.ResultCode)
	}
	if resp.CheckoutRequestID != "ws_CO_12345" {
		t.Errorf("expected CheckoutRequestID ws_CO_12345, got %s", resp.CheckoutRequestID)
	}
}

func TestSTKPushQuery_Validations(t *testing.T) {
	client := &Client{
		baseURL:      "http://localhost",
		httpClient:   testHTTPClient(http.DefaultClient),
		passkey:      "test-passkey",
		tokenManager: fakeTokenProvider{},
	}

	// Missing checkout ID
	_, err := client.STKPushQuery(context.Background(), STKPushQueryRequest{
		BusinessShortCode: "174379",
	})
	if err == nil {
		t.Error("expected error for missing checkout request ID")
	}

	// Missing short code
	_, err = client.STKPushQuery(context.Background(), STKPushQueryRequest{
		CheckoutRequestID: "ws_CO_12345",
	})
	if err == nil {
		t.Error("expected error for missing short code")
	}
}

func TestSTKPushQuery_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		passkey:      "test-passkey",
		tokenManager: fakeTokenProvider{},
	}

	_, err := client.STKPushQuery(context.Background(), STKPushQueryRequest{
		BusinessShortCode: "174379",
		CheckoutRequestID: "ws_CO_12345",
	})
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}
