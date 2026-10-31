package mpesa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestB2CPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/mpesa/b2c/v1/paymentrequest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		var req B2CRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Amount != 500 {
			t.Errorf("expected amount 500, got %d", req.Amount)
		}
		if req.PartyB != "254712345678" {
			t.Errorf("expected PartyB 254712345678, got %s", req.PartyB)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"ConversationID": "AG_20230805_00004f8",
			"OriginatorConversationID": "1234-5678",
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

	resp, err := client.B2CPayment(context.Background(), B2CRequest{
		InitiatorName:      "test_initiator",
		SecurityCredential: "test_credential",
		PartyA:             "600000",
		PartyB:             "254712345678",
		Amount:             500,
		Remarks:            "Salary",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("expected ResponseCode 0, got %s", resp.ResponseCode)
	}
	if resp.ConversationID != "AG_20230805_00004f8" {
		t.Errorf("unexpected ConversationID: %s", resp.ConversationID)
	}
}

func TestB2CPayment_Validations(t *testing.T) {
	client := &Client{
		baseURL:      "http://localhost",
		httpClient:   testHTTPClient(http.DefaultClient),
		tokenManager: fakeTokenProvider{},
	}

	tests := []struct {
		name string
		req  B2CRequest
	}{
		{
			name: "missing initiator",
			req: B2CRequest{
				SecurityCredential: "cred",
				PartyA:             "600000",
				PartyB:             "254712345678",
				Amount:             100,
			},
		},
		{
			name: "missing credential",
			req: B2CRequest{
				InitiatorName: "init",
				PartyA:        "600000",
				PartyB:        "254712345678",
				Amount:        100,
			},
		},
		{
			name: "missing partyA",
			req: B2CRequest{
				InitiatorName:      "init",
				SecurityCredential: "cred",
				PartyB:             "254712345678",
				Amount:             100,
			},
		},
		{
			name: "missing partyB",
			req: B2CRequest{
				InitiatorName:      "init",
				SecurityCredential: "cred",
				PartyA:             "600000",
				Amount:             100,
			},
		},
		{
			name: "zero amount",
			req: B2CRequest{
				InitiatorName:      "init",
				SecurityCredential: "cred",
				PartyA:             "600000",
				PartyB:             "254712345678",
				Amount:             0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.B2CPayment(context.Background(), tt.req)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestB2CPayment_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := &Client{
		baseURL:      server.URL,
		httpClient:   testHTTPClient(server.Client()),
		tokenManager: fakeTokenProvider{},
	}

	_, err := client.B2CPayment(context.Background(), B2CRequest{
		InitiatorName:      "init",
		SecurityCredential: "cred",
		PartyA:             "600000",
		PartyB:             "254712345678",
		Amount:             100,
	})
	if err == nil {
		t.Error("expected error on HTTP 502")
	}
}
