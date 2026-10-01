package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestB2CPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/mpesa/b2c/v1/paymentrequest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_token" {
			t.Errorf("expected bearer token, got %s", auth)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content type")
		}

		var requestBody B2CRequest

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			t.Fatalf("failed decoding request body: %v", err)
		}

		if requestBody.Amount != 100 {
			t.Errorf("expected amount 100, got %d", requestBody.Amount)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"ConversationID":"conv123",
			"OriginatorConversationID":"orig123",
			"ResponseCode":"0",
			"ResponseDescription":"Accept the service request successfully"
		}`))
	}))

	defer server.Close()

	tokenManager := NewTokenManager(
		server.Client(),
		"key",
		"secret",
		server.URL,
	)

	// Use a cached token so the test focuses on B2C logic.
	tokenManager.accessToken = "test_token"
	tokenManager.expiry = time.Now().Add(time.Hour)

	client := NewMpesaClient(
		server.URL,
		server.Client(),
		"",
		tokenManager,
	)

	response, err := client.B2CPayment(
		context.Background(),
		&B2CRequest{
			InitiatorName: "test",
			CommandID:     "BusinessPayment",
			Amount:        100,
			PartyA:        "600000",
			PartyB:        "254700000000",
			Remarks:       "test payment",
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !response.Success {
		t.Error("expected successful payment")
	}

	if response.TransactionID != "conv123" {
		t.Errorf(
			"expected transaction id conv123, got %s",
			response.TransactionID,
		)
	}

	if response.ErrorCode != "0" {
		t.Errorf(
			"expected error code 0, got %s",
			response.ErrorCode,
		)
	}
}

func TestB2CPayment_NoTokenManager(t *testing.T) {

	client := NewMpesaClient(
		"http://localhost",
		nil,
		"",
		nil,
	)

	_, err := client.B2CPayment(
		context.Background(),
		&B2CRequest{},
	)

	if err == nil {
		t.Error("expected error when token manager is missing")
	}
}

func TestB2CPayment_FailedTransaction(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{
			"ConversationID":"failed123",
			"ResponseCode":"1",
			"ResponseDescription":"Failed"
		}`))
	}))

	defer server.Close()

	tokenManager := NewTokenManager(
		server.Client(),
		"key",
		"secret",
		server.URL,
	)

	tokenManager.accessToken = "test_token"
	tokenManager.expiry = time.Now().Add(time.Hour)

	client := NewMpesaClient(
		server.URL,
		server.Client(),
		"",
		tokenManager,
	)

	response, err := client.B2CPayment(
		context.Background(),
		&B2CRequest{},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Success {
		t.Error("expected failed transaction")
	}

	if response.ErrorCode != "1" {
		t.Errorf("expected error code 1, got %s", response.ErrorCode)
	}
}
