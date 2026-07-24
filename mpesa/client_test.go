package mpesa

import (
	"net/http/httptest"
	"testing"
	"context"
	"encoding/json"
	
	"net/http"
)

func TestSTKPushUsesCorrectendpoint(t *testing.T) {
	var (
		method string
		path string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		method = r.Method
		path  = r.URL.Path

		w.Header().Set("Content-Type", "application/json")

		w.Write([]byte(`{
		"SellerRequestID": "123",
		"OutRequestID":"456",
		"Response":"0",
		"Description":"Success",
		"Message": "Accepted"
		}`))
	}))

	defer server.Close()


	client := &Client{
		baseURL: server.URL,
		httpClient: server.Client(),
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

	_,err := client.STKPush(context.Background(), req)

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

func TestSTKPushParsesSuccessResponse(t *testing.T){
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		w.Write([]byte(`{
			"MerchantRequestID":"123",
			"CheckoutRequestID":"456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
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

	resp,err := client.STKPush(context.Background(), req)

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
	var idempotencyKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idempotencyKey = r.Header.Get("Idempotency-Key")

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"MerchantRequestID":"123",
			"CheckoutRequestID":"456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))
	}))

	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	req :=  STKPushRequest{
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

	_,err := client.STKPush(context.Background(), req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if idempotencyKey != "stk-push-001" {
		t.Errorf("expected Idempotency-Key header %q, got %q", "stk-push-001", idempotencyKey)
	}

	// if idempotencyKey != "stk-push-001" {
	// 	t.Errorf("expected Idempotency-Key header %q, got %q", "stk-push-001", idempotencyKey)
	// }
}

func TestSTKPushGeneratesTimestampPassword(t *testing.T) {
	var received STKPushRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}


		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"MerchantRequestID":"123",
			"CheckoutRequestID":"456",
			"ResponseCode":"0",
			"ResponseDescription":"Success",
			"CustomerMessage":"Accepted"
		}`))

	}))

	defer server.Close()

	client := &Client{
		baseURL: server.URL,
		httpClient : server.Client(),
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

	_,err := client.STKPush(context.Background(), req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Timestamp == "" {
		t.Errorf("expected TimeStamp to have legth 14 got %d", len(received.Timestamp))
	}

	if received.Password == ""{
		t.Errorf("expected password to be generated")
	}

}