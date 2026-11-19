package mpesa

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Flying-Tea-Squad/paykit-go"
	"github.com/stretchr/testify/require"
)

func TestNew_Validations(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing consumer key",
			cfg:     Config{ConsumerSecret: "sec", Passkey: "pass", BusinessShortCode: "174379"},
			wantErr: "consumer key is required",
		},
		{
			name:    "missing consumer secret",
			cfg:     Config{ConsumerKey: "key", Passkey: "pass", BusinessShortCode: "174379"},
			wantErr: "consumer secret is required",
		},
		{
			name:    "missing passkey",
			cfg:     Config{ConsumerKey: "key", ConsumerSecret: "sec", BusinessShortCode: "174379"},
			wantErr: "passkey is required",
		},
		{
			name:    "missing short code",
			cfg:     Config{ConsumerKey: "key", ConsumerSecret: "sec", Passkey: "pass"},
			wantErr: "business short code is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw, err := New(tt.cfg)
			if gw != nil {
				t.Fatalf("expected nil gateway, got %v", gw)
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}

	// ShortCode alias fallback
	gw, err := New(Config{
		ConsumerKey:    "key",
		ConsumerSecret: "sec",
		Passkey:        "pass",
		ShortCode:      "174379",
	})
	if err != nil {
		t.Fatalf("unexpected error using ShortCode alias: %v", err)
	}
	if gw.cfg.BusinessShortCode != "174379" {
		t.Errorf("expected BusinessShortCode 174379, got %s", gw.cfg.BusinessShortCode)
	}

	// Environment URL defaults
	gwSandbox, _ := New(Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		Environment:       EnvironmentSandbox,
	})
	if gwSandbox.cfg.BaseURL != "https://sandbox.safaricom.co.ke" {
		t.Errorf("expected sandbox URL, got %s", gwSandbox.cfg.BaseURL)
	}

	gwProd, _ := New(Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		Environment:       EnvironmentProduction,
	})
	if gwProd.cfg.BaseURL != "https://api.safaricom.co.ke" {
		t.Errorf("expected production URL, got %s", gwProd.cfg.BaseURL)
	}
}

func TestNewWithClient(t *testing.T) {
	_, err := NewWithClient(nil, Config{})
	if err == nil {
		t.Error("expected error with nil client")
	}

	client := &Client{}
	gw, err := NewWithClient(client, Config{ShortCode: "174379"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.Client() != client {
		t.Errorf("expected client %v, got %v", client, gw.Client())
	}
	if gw.cfg.BusinessShortCode != "174379" {
		t.Errorf("expected ShortCode populated, got %s", gw.cfg.BusinessShortCode)
	}
}

func TestGateway_Name(t *testing.T) {
	gw := &Gateway{}
	if gw.Name() != "mpesa" {
		t.Errorf("expected name mpesa, got %s", gw.Name())
	}
}

func TestGateway_Charge_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mpesa/stkpush/v1/processrequest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req STKPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if req.Amount != 50 {
			t.Errorf("expected amount 50 KES, got %d", req.Amount)
		}
		if req.PhoneNumber != "254712345678" {
			t.Errorf("expected normalized phone 254712345678, got %s", req.PhoneNumber)
		}
		if req.AccountReference != "INV-001" {
			t.Errorf("expected reference INV-001, got %s", req.AccountReference)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"MerchantRequestID": "29115-34620561-1",
			"CheckoutRequestID": "ws_CO_19122026100122",
			"ResponseCode": "0",
			"ResponseDescription": "Success. Request accepted for processing",
			"CustomerMessage": "Success. Request accepted for processing"
		}`))
	}))
	defer server.Close()

	gw, err := New(Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		BaseURL:           server.URL,
		CallbackURL:       "https://example.com/callback",
		HTTPClient:        testHTTPClient(server.Client()),
	})
	if err != nil {
		t.Fatalf("failed to create gateway: %v", err)
	}
	gw.client.tokenManager = fakeTokenProvider{}

	resp, err := gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount:      paykit.NewKES(50),
		Phone:       "0712345678",
		Reference:   "INV-001",
		Description: "Order payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Status != paykit.StatusPendingAction {
		t.Errorf("expected StatusPendingAction, got %s", resp.Status)
	}
	if resp.CheckoutRequestID != "ws_CO_19122026100122" {
		t.Errorf("unexpected CheckoutRequestID: %s", resp.CheckoutRequestID)
	}
	if resp.TransactionID != "ws_CO_19122026100122" {
		t.Errorf("unexpected TransactionID: %s", resp.TransactionID)
	}
	if resp.Message != "Success. Request accepted for processing" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestGateway_Charge_Validations(t *testing.T) {
	gw := &Gateway{
		cfg: Config{
			BusinessShortCode: "174379",
			CallbackURL:       "https://example.com/callback",
		},
	}

	// Nil request
	_, err := gw.Charge(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("expected nil request error, got %v", err)
	}

	// Unsupported currency
	_, err = gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount: paykit.NewMoney(1000, "USD"),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported currency") {
		t.Errorf("expected unsupported currency error, got %v", err)
	}

	// Zero amount
	_, err = gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount: paykit.NewKES(0),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Errorf("expected zero amount error, got %v", err)
	}

	// Sub-shilling amount (< 100 minor units / 1 KES)
	_, err = gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount: paykit.NewMoney(50, "KES"),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "at least 1 KES") {
		t.Errorf("expected sub-shilling error, got %v", err)
	}

	// Sub-shilling remainder cents (e.g. 150 cents = 1.50 KES)
	_, err = gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount: paykit.NewMoney(150, "KES"),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "sub-shilling amounts are not supported") {
		t.Errorf("expected sub-shilling remainder error, got %v", err)
	}

	// Invalid phone number
	_, err = gw.Charge(context.Background(), &paykit.ChargeRequest{
		Amount: paykit.NewKES(10),
		Phone:  "12345",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid phone number") {
		t.Errorf("expected invalid phone error, got %v", err)
	}

	// Missing callback URL
	gwNoCallback := &Gateway{
		cfg: Config{BusinessShortCode: "174379"},
	}
	_, err = gwNoCallback.Charge(context.Background(), &paykit.ChargeRequest{
		Amount: paykit.NewKES(10),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "callback URL is required") {
		t.Errorf("expected missing callback URL error, got %v", err)
	}
}

func TestGateway_QueryStatus(t *testing.T) {
	tests := []struct {
		name         string
		responseCode string
		resultCode   string
		wantStatus   paykit.TransactionStatus
		wantSucc     bool
	}{
		{
			name:         "success",
			responseCode: "0",
			resultCode:   "0",
			wantStatus:   paykit.StatusSuccess,
			wantSucc:     true,
		},
		{
			name:         "cancelled by user",
			responseCode: "0",
			resultCode:   "1032",
			wantStatus:   paykit.StatusCancelled,
			wantSucc:     true,
		},
		{
			name:         "timeout",
			responseCode: "0",
			resultCode:   "1037",
			wantStatus:   paykit.StatusTimeout,
			wantSucc:     true,
		},
		{
			name:         "failed",
			responseCode: "0",
			resultCode:   "1",
			wantStatus:   paykit.StatusFailed,
			wantSucc:     true,
		},
		{
			name:         "query error from daraja",
			responseCode: "1",
			resultCode:   "1",
			wantStatus:   paykit.StatusFailed,
			wantSucc:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"ResponseCode": "` + tt.responseCode + `",
					"ResponseDescription": "Accepted",
					"MerchantRequestID": "merch-1",
					"CheckoutRequestID": "ws_CO_999",
					"ResultCode": "` + tt.resultCode + `",
					"ResultDesc": "Status message"
				}`))
			}))
			defer server.Close()

			gw := &Gateway{
				client: &Client{
					baseURL:      server.URL,
					httpClient:   testHTTPClient(server.Client()),
					passkey:      "pass",
					tokenManager: fakeTokenProvider{},
				},
				cfg: Config{BusinessShortCode: "174379"},
			}

			resp, err := gw.QueryStatus(context.Background(), &paykit.StatusRequest{
				TransactionID: "ws_CO_999",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, resp.Status)
			}
			if resp.Success != tt.wantSucc {
				t.Errorf("expected success %v, got %v", tt.wantSucc, resp.Success)
			}
			if resp.TransactionID != "ws_CO_999" {
				t.Errorf("expected TransactionID ws_CO_999, got %s", resp.TransactionID)
			}
		})
	}

	// Missing checkout ID error
	gw := &Gateway{}
	_, err := gw.QueryStatus(context.Background(), &paykit.StatusRequest{})
	if err == nil || !strings.Contains(err.Error(), "transaction ID or checkout_request_id is required") {
		t.Errorf("expected missing ID error, got %v", err)
	}

	// Nil request error
	_, err = gw.QueryStatus(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("expected nil request error, got %v", err)
	}
}

func TestGateway_Refund_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"ConversationID": "AG_REV_123",
			"OriginatorConversationID": "ORIG_REV_123",
			"ResponseCode": "0",
			"ResponseDescription": "Accept the service request successfully"
		}`))
	}))
	defer server.Close()

	gw := &Gateway{
		client: &Client{
			baseURL:      server.URL,
			httpClient:   testHTTPClient(server.Client()),
			tokenManager: fakeTokenProvider{},
		},
		cfg: Config{
			BusinessShortCode:  "174379",
			InitiatorName:      "test_initiator",
			SecurityCredential: "test_credential",
		},
	}

	resp, err := gw.Refund(context.Background(), &paykit.RefundRequest{
		TransactionID: "LK12345678",
		Amount:        paykit.NewKES(100),
		Reason:        "Customer return",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Status != paykit.StatusProcessing {
		t.Errorf("expected StatusProcessing, got %s", resp.Status)
	}
	if resp.RefundID != "AG_REV_123" {
		t.Errorf("expected RefundID AG_REV_123, got %s", resp.RefundID)
	}
	if resp.TransactionID != "LK12345678" {
		t.Errorf("expected TransactionID LK12345678, got %s", resp.TransactionID)
	}
}

func TestGateway_Refund_Validations(t *testing.T) {
	gw := &Gateway{}

	// Nil request
	_, err := gw.Refund(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("expected nil error, got %v", err)
	}

	// Missing transaction ID
	_, err = gw.Refund(context.Background(), &paykit.RefundRequest{})
	if err == nil || !strings.Contains(err.Error(), "transaction ID is required") {
		t.Errorf("expected missing ID error, got %v", err)
	}

	// Missing amount (zero amount)
	_, err = gw.Refund(context.Background(), &paykit.RefundRequest{TransactionID: "LK123"})
	if err == nil || !strings.Contains(err.Error(), "refund amount is required") {
		t.Errorf("expected missing amount error, got %v", err)
	}

	// Amount less than 1 KES
	_, err = gw.Refund(context.Background(), &paykit.RefundRequest{
		TransactionID: "LK123",
		Amount:        paykit.NewMoney(50, "KES"),
	})
	if err == nil || !strings.Contains(err.Error(), "at least 1 KES") {
		t.Errorf("expected sub-shilling min error, got %v", err)
	}

	// Sub-shilling remainder
	_, err = gw.Refund(context.Background(), &paykit.RefundRequest{
		TransactionID: "LK123",
		Amount:        paykit.NewMoney(150, "KES"),
	})
	if err == nil || !strings.Contains(err.Error(), "sub-shilling amounts are not supported") {
		t.Errorf("expected sub-shilling remainder error, got %v", err)
	}

	// Missing credentials
	gwWithID := &Gateway{}
	_, err = gwWithID.Refund(context.Background(), &paykit.RefundRequest{
		TransactionID: "LK123",
		Amount:        paykit.NewKES(10),
	})
	if err == nil || !strings.Contains(err.Error(), "initiator is required") {
		t.Errorf("expected missing initiator error, got %v", err)
	}
}

func TestGateway_Disburse_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"ConversationID": "AG_B2C_123",
			"OriginatorConversationID": "ORIG_B2C_123",
			"ResponseCode": "0",
			"ResponseDescription": "Accept the service request successfully"
		}`))
	}))
	defer server.Close()

	gw := &Gateway{
		client: &Client{
			baseURL:      server.URL,
			httpClient:   testHTTPClient(server.Client()),
			tokenManager: fakeTokenProvider{},
		},
		cfg: Config{
			BusinessShortCode:  "600000",
			InitiatorName:      "test_initiator",
			SecurityCredential: "test_credential",
		},
	}

	resp, err := gw.Disburse(context.Background(), &paykit.DisbursementRequest{
		Amount:      paykit.NewKES(200),
		Phone:       "+254712345678",
		Description: "Vendor payout",
		Reference:   "PAYOUT-01",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Status != paykit.StatusProcessing {
		t.Errorf("expected StatusProcessing, got %s", resp.Status)
	}
	if resp.TransactionID != "AG_B2C_123" {
		t.Errorf("expected TransactionID AG_B2C_123, got %s", resp.TransactionID)
	}
}

func TestGateway_Disburse_Validations(t *testing.T) {
	gw := &Gateway{
		cfg: Config{
			BusinessShortCode:  "600000",
			InitiatorName:      "init",
			SecurityCredential: "cred",
		},
	}

	// Nil request
	_, err := gw.Disburse(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("expected nil error, got %v", err)
	}

	// Zero amount
	_, err = gw.Disburse(context.Background(), &paykit.DisbursementRequest{
		Amount: paykit.NewKES(0),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Errorf("expected zero amount error, got %v", err)
	}

	// Invalid phone
	_, err = gw.Disburse(context.Background(), &paykit.DisbursementRequest{
		Amount: paykit.NewKES(10),
		Phone:  "invalid-phone",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid recipient phone") {
		t.Errorf("expected invalid phone error, got %v", err)
	}

	// Sub-shilling remainder
	_, err = gw.Disburse(context.Background(), &paykit.DisbursementRequest{
		Amount: paykit.NewMoney(150, "KES"),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "sub-shilling amounts are not supported") {
		t.Errorf("expected sub-shilling remainder error, got %v", err)
	}

	// Missing credentials
	gwNoCreds := &Gateway{}
	_, err = gwNoCreds.Disburse(context.Background(), &paykit.DisbursementRequest{
		Amount: paykit.NewKES(10),
		Phone:  "0712345678",
	})
	if err == nil || !strings.Contains(err.Error(), "initiator name is required") {
		t.Errorf("expected missing initiator error, got %v", err)
	}
}

// --- Auto-generated SecurityCredential tests ---

func TestNew_AutoGenerateSecurityCredential(t *testing.T) {
	privateKey, certPEM := generateTestCertificateForGateway(t)
	password := "TestInitiatorPassword123"

	cfg := Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		InitiatorPassword: password,
		CertificatePEM:    certPEM,
	}

	gw, err := New(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, gw.cfg.SecurityCredential, "SecurityCredential should be auto-generated")

	// Verify the generated credential can be decrypted back to the password
	ciphertext, err := base64.StdEncoding.DecodeString(gw.cfg.SecurityCredential)
	require.NoError(t, err)
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	require.NoError(t, err)
	require.Equal(t, password, string(decrypted))
}

func TestNew_NoAutoGenerate_WhenSecurityCredentialProvided(t *testing.T) {
	_, certPEM := generateTestCertificateForGateway(t)
	password := "TestInitiatorPassword123"
	explicitCred := "explicitly-provided-credential"

	cfg := Config{
		ConsumerKey:        "key",
		ConsumerSecret:     "sec",
		Passkey:            "pass",
		BusinessShortCode:  "174379",
		SecurityCredential: explicitCred,
		InitiatorPassword:  password,
		CertificatePEM:     certPEM,
	}

	gw, err := New(cfg)
	require.NoError(t, err)
	require.Equal(t, explicitCred, gw.cfg.SecurityCredential, "explicit SecurityCredential should take precedence")
}

func TestNew_Error_WhenOnlyPasswordProvided(t *testing.T) {
	cfg := Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		InitiatorPassword: "password",
		// CertificatePEM omitted
	}

	_, err := New(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "both InitiatorPassword and CertificatePEM are required")
}

func TestNew_Error_WhenOnlyCertProvided(t *testing.T) {
	_, certPEM := generateTestCertificateForGateway(t)

	cfg := Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		CertificatePEM:    certPEM,
		// InitiatorPassword omitted
	}

	_, err := New(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "both InitiatorPassword and CertificatePEM are required")
}

func TestNew_AutoGenerateErrorPropagates(t *testing.T) {
	// Invalid PEM (not a valid certificate)
	certPEM := []byte(`-----BEGIN CERTIFICATE-----
invalid
-----END CERTIFICATE-----`)

	cfg := Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
		InitiatorPassword: "password",
		CertificatePEM:    certPEM,
	}

	_, err := New(cfg)
	require.Error(t, err)
	// Should wrap the sentinel error
	require.ErrorContains(t, err, "failed to generate security credential")
}

func TestNew_ConfigNotMutated(t *testing.T) {
	_, certPEM := generateTestCertificateForGateway(t)
	password := "TestInitiatorPassword123"

	originalCfg := Config{
		ConsumerKey:        "key",
		ConsumerSecret:     "sec",
		Passkey:            "pass",
		BusinessShortCode:  "174379",
		InitiatorPassword:  password,
		CertificatePEM:     certPEM,
		SecurityCredential: "",
	}

	// Make a copy for comparison
	cfgCopy := originalCfg

	_, err := New(originalCfg)
	require.NoError(t, err)

	// Original config should be unchanged (passed by value)
	require.Equal(t, cfgCopy.SecurityCredential, originalCfg.SecurityCredential)
	require.Equal(t, cfgCopy.InitiatorPassword, originalCfg.InitiatorPassword)
	require.Equal(t, cfgCopy.CertificatePEM, originalCfg.CertificatePEM)
}

func TestNew_STKPushOnly_NoCredentialNeeded(t *testing.T) {
	// Config without any credential fields should succeed (STK Push only)
	cfg := Config{
		ConsumerKey:       "key",
		ConsumerSecret:    "sec",
		Passkey:           "pass",
		BusinessShortCode: "174379",
	}

	gw, err := New(cfg)
	require.NoError(t, err)
	require.Empty(t, gw.cfg.SecurityCredential)
	require.Empty(t, gw.cfg.InitiatorPassword)
	require.Nil(t, gw.cfg.CertificatePEM)
}

func TestNew_SecurityCredentialPrecedence_ExplicitOverAuto(t *testing.T) {
	_, certPEM := generateTestCertificateForGateway(t)
	password := "TestInitiatorPassword123"
	explicitCred := "explicit-credential"

	cfg := Config{
		ConsumerKey:        "key",
		ConsumerSecret:     "sec",
		Passkey:            "pass",
		BusinessShortCode:  "174379",
		SecurityCredential: explicitCred,
		InitiatorPassword:  password,
		CertificatePEM:     certPEM,
	}

	gw, err := New(cfg)
	require.NoError(t, err)
	require.Equal(t, explicitCred, gw.cfg.SecurityCredential)
}

func TestNew_SecurityCredentialPrecedence_ExplicitWithMalformedCert(t *testing.T) {
	// Malformed cert should be ignored when explicit SecurityCredential provided
	malformedCert := []byte(`-----BEGIN CERTIFICATE-----
not a valid cert
-----END CERTIFICATE-----`)

	explicitCred := "explicit-credential"

	cfg := Config{
		ConsumerKey:        "key",
		ConsumerSecret:     "sec",
		Passkey:            "pass",
		BusinessShortCode:  "174379",
		SecurityCredential: explicitCred,
		InitiatorPassword:  "password",
		CertificatePEM:     malformedCert,
	}

	gw, err := New(cfg)
	require.NoError(t, err, "malformed cert should be ignored when SecurityCredential is explicit")
	require.Equal(t, explicitCred, gw.cfg.SecurityCredential)
}

// Test helper for gateway tests
func generateTestCertificateForGateway(t *testing.T) (*rsa.PrivateKey, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test CA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return privateKey, certPEM
}
