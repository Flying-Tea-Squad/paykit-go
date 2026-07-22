package mpesa

import (
	"net/http/httptest"
	"testing"
	"context"
	
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