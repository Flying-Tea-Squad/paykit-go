package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReverseTransaction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/mpesa/reversal/v1/request" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		var req ReversalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.TransactionID != "LK12345678" {
			t.Errorf("unexpected TransactionID: %s", req.TransactionID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"ConversationID": "AG_REV_12345",
			"OriginatorConversationID": "ORIG_REV_12345",
			"ResponseCode": "0",
			"ResponseDescription": "Accept the service request successfully"
		}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		tokenManager: fakeTokenProvider{},
	}

	resp, err := client.ReverseTransaction(context.Background(), ReversalRequest{
		Initiator:          "test_initiator",
		SecurityCredential: "test_credential",
		TransactionID:      "LK12345678",
		Amount:             250,
		ReceiverParty:      "600000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("expected ResponseCode 0, got %s", resp.ResponseCode)
	}
	if resp.ConversationID != "AG_REV_12345" {
		t.Errorf("unexpected ConversationID: %s", resp.ConversationID)
	}
}

func TestReverseTransaction_Validations(t *testing.T) {
	client := &Client{
		baseURL:      "http://localhost",
		httpClient:   testHTTPClient(http.DefaultClient),
		tokenManager: fakeTokenProvider{},
	}

	// Missing initiator
	_, err := client.ReverseTransaction(context.Background(), ReversalRequest{
		SecurityCredential: "cred",
		TransactionID:      "LK123",
		ReceiverParty:      "600000",
		Amount:             100,
	})
	if err == nil {
		t.Error("expected error for missing initiator")
	}

	// Missing transaction ID
	_, err = client.ReverseTransaction(context.Background(), ReversalRequest{
		Initiator:          "init",
		SecurityCredential: "cred",
		ReceiverParty:      "600000",
		Amount:             100,
	})
	if err == nil {
		t.Error("expected error for missing transaction ID")
	}
}
