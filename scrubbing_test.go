package paykit_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskPhone_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 12-digit Kenyan E.164 without plus
		{
			name:     "Kenyan 12-digit 2547 standard (Issue #33 reference)",
			input:    "254712345678",
			expected: "2547***5678",
		},
		{
			name:     "Kenyan 12-digit 2541 Airtel",
			input:    "254112345678",
			expected: "2541***5678",
		},
		{
			name:     "Kenyan 12-digit 2547 with trailing zeros",
			input:    "254700000001",
			expected: "2547***0001",
		},

		// 13-character Kenyan full E.164 with plus
		{
			name:     "Kenyan full E.164 with plus 2547",
			input:    "+254712345678",
			expected: "+2547***5678",
		},
		{
			name:     "Kenyan full E.164 with plus 2541",
			input:    "+254112345678",
			expected: "+2541***5678",
		},

		// 10-digit local Kenyan format with leading zero
		{
			name:     "Local Kenyan 07 Safaricom",
			input:    "0712345678",
			expected: "07***5678",
		},
		{
			name:     "Local Kenyan 01 Airtel",
			input:    "0112345678",
			expected: "01***5678",
		},

		// Formatted phone numbers with spaces, hyphens, and dots
		{
			name:     "Local Kenyan with hyphens",
			input:    "0712-345-678",
			expected: "07***5678",
		},
		{
			name:     "Local Kenyan with spaces",
			input:    "0712 345 678",
			expected: "07***5678",
		},
		{
			name:     "Full E.164 with spaces",
			input:    "+254 712 345 678",
			expected: "+2547***5678",
		},
		{
			name:     "E.164 with hyphens",
			input:    "254-712-345-678",
			expected: "2547***5678",
		},

		// Other East African / International MSISDNs
		{
			name:     "Uganda Airtel 2567",
			input:    "+256712345678",
			expected: "+2567***5678",
		},
		{
			name:     "Tanzania 2557",
			input:    "255712345678",
			expected: "2557***5678",
		},

		// Shortcodes and diagnostic IDs preserved (not masked)
		{
			name:     "Safaricom shortcode 6 digits",
			input:    "174379",
			expected: "174379",
		},
		{
			name:     "Paybill 6 digits",
			input:    "600982",
			expected: "600982",
		},
		{
			name:     "Till number 5 digits",
			input:    "29402",
			expected: "29402",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Alphanumeric non-phone",
			input:    "OrderRef123",
			expected: "OrderRef123",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := paykit.MaskPhone(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRegexScrubber_TableDriven(t *testing.T) {
	t.Parallel()

	scrubber := paykit.NewDefaultScrubber()
	require.NotNil(t, scrubber)

	tests := []struct {
		name        string
		input       string
		mustContain []string
		mustExclude []string
	}{
		// Authorization & API Key headers
		{
			name:        "Authorization Bearer header in wire dump",
			input:       "Authorization: Bearer cFJZc2FmYXJpY29tMTIzNDU2Nzg=\r\nContent-Type: application/json",
			mustContain: []string{"Authorization: Bearer [REDACTED]", "Content-Type: application/json"},
			mustExclude: []string{"cFJZc2FmYXJpY29tMTIzNDU2Nzg="},
		},
		{
			name:        "Authorization Basic header in wire dump",
			input:       "Authorization: Basic dXNlcjpwYXNzd29yZDEyMw==\r\nHost: api.example.com",
			mustContain: []string{"Authorization: Basic [REDACTED]", "Host: api.example.com"},
			mustExclude: []string{"dXNlcjpwYXNzd29yZDEyMw=="},
		},
		{
			name:        "Proxy-Authorization Basic header",
			input:       "Proxy-Authorization: Basic cHJveHl1c2VyOnByb3h5cGFzcw==\r\n",
			mustContain: []string{"Proxy-Authorization: Basic [REDACTED]"},
			mustExclude: []string{"cHJveHl1c2VyOnByb3h5cGFzcw=="},
		},
		{
			name:        "Custom scheme Authorization header",
			input:       "Authorization: Token secret_api_token_12345\r\n",
			mustContain: []string{"Authorization: [REDACTED]"},
			mustExclude: []string{"secret_api_token_12345"},
		},
		{
			name:        "X-Api-Key header",
			input:       "X-Api-Key: pesapal_live_sec_abcdef987654321\r\n",
			mustContain: []string{"X-Api-Key: [REDACTED]"},
			mustExclude: []string{"pesapal_live_sec_abcdef987654321"},
		},
		{
			name:        "Api-Key header without X- prefix",
			input:       "Api-Key: live_api_key_xyz123\r\n",
			mustContain: []string{"Api-Key: [REDACTED]"},
			mustExclude: []string{"live_api_key_xyz123"},
		},

		// Standalone Bearer / Basic tokens
		{
			name:        "Standalone Bearer in log message",
			input:       "OAuth token received: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token123",
			mustContain: []string{"OAuth token received: Bearer [REDACTED]"},
			mustExclude: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token123"},
		},
		{
			name:        "Standalone Basic in log message",
			input:       "Calculated basic auth header Basic dGVzdHVzZXI6dGVzdHBhc3M=",
			mustContain: []string{"Calculated basic auth header Basic [REDACTED]"},
			mustExclude: []string{"dGVzdHVzZXI6dGVzdHBhc3M="},
		},

		// Customer Phone Numbers (MSISDNs)
		{
			name:        "Standalone 12-digit Kenyan phone in text (Issue #33)",
			input:       "Sending STK push prompt to 254712345678 now",
			mustContain: []string{"Sending STK push prompt to 2547***5678 now"},
			mustExclude: []string{"254712345678"},
		},
		{
			name:        "Standalone +2547 full E.164 in text",
			input:       "Callback delivered for +254712345678 successfully",
			mustContain: []string{"Callback delivered for +2547***5678 successfully"},
			mustExclude: []string{"+254712345678"},
		},
		{
			name:        "Standalone 2541 Airtel phone in text",
			input:       "Airtel Money disbursement to 254112345678 queued",
			mustContain: []string{"Airtel Money disbursement to 2541***5678 queued"},
			mustExclude: []string{"254112345678"},
		},
		{
			name:        "Standalone 10-digit 07 phone in text",
			input:       "Customer entered phone 0712345678 at checkout",
			mustContain: []string{"Customer entered phone 07***5678 at checkout"},
			mustExclude: []string{"0712345678"},
		},
		{
			name:        "Multiple phone numbers in a single line",
			input:       "Transfer from 254712345678 to 254798765432 completed",
			mustContain: []string{"Transfer from 2547***5678 to 2547***5432 completed"},
			mustExclude: []string{"254712345678", "254798765432"},
		},

		// JSON Payloads (M-Pesa STK Push)
		{
			name: "M-Pesa STK Push request payload",
			input: `{"BusinessShortCode":"174379","Password":"raw-daraja-password-xyz","Timestamp":"20261101235959",` +
				`"TransactionType":"CustomerPayBillOnline","Amount":100,"PartyA":"254712345678","PartyB":"174379",` +
				`"PhoneNumber":"254712345678","CallBackURL":"https://example.com/cb","AccountReference":"INV001"}`,
			mustContain: []string{
				`"BusinessShortCode":"174379"`,
				`"Password":"[REDACTED]"`,
				`"Timestamp":"20261101235959"`,
				`"Amount":100`,
				`"PartyA":"2547***5678"`,
				`"PartyB":"174379"`,
				`"PhoneNumber":"2547***5678"`,
				`"CallBackURL":"https://example.com/cb"`,
				`"AccountReference":"INV001"`,
			},
			mustExclude: []string{
				"raw-daraja-password-xyz",
				"254712345678",
			},
		},
		{
			name:        "JSON with unquoted numeric PartyA phone",
			input:       `{"PartyA":254712345678,"Amount":50}`,
			mustContain: []string{`"PartyA":"2547***5678"`, `"Amount":50`},
			mustExclude: []string{"254712345678"},
		},
		{
			name:        "JSON with empty string phone preserves valid syntax",
			input:       `{"phone": "","status":"active"}`,
			mustContain: []string{`{"phone": "","status":"active"}`},
		},
		{
			name:        "JSON with null phone preserves null literal",
			input:       `{"phone": null,"status":"active"}`,
			mustContain: []string{`{"phone": null,"status":"active"}`},
			mustExclude: []string{`"null"`},
		},
		{
			name:        "JSON with nested phone object preserves structure while masking inner number",
			input:       `{"phone": {"number": "254712345678"}}`,
			mustContain: []string{`{"phone": {"number": "2547***5678"}}`},
			mustExclude: []string{"254712345678"},
		},

		// JSON Credentials: Passkey, ConsumerSecret, Tokens
		{
			name:        "JSON with passkey",
			input:       `{"passkey":"bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"}`,
			mustContain: []string{`"passkey":"[REDACTED]"`},
			mustExclude: []string{"bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"},
		},
		{
			name:        "JSON Pesapal auth credentials",
			input:       `{"consumer_key":"my-pesapal-key","consumer_secret":"pesapal-secret-987"}`,
			mustContain: []string{`"consumer_key":"my-pesapal-key"`, `"consumer_secret":"[REDACTED]"`},
			mustExclude: []string{"pesapal-secret-987"},
		},
		{
			name:        "JSON client_secret",
			input:       `{"client_id":"airtel-client","client_secret":"airtel-sec-007"}`,
			mustContain: []string{`"client_id":"airtel-client"`, `"client_secret":"[REDACTED]"`},
			mustExclude: []string{"airtel-sec-007"},
		},
		{
			name:        "JSON token and access_token",
			input:       `{"access_token":"token-abc123xyz","token_type":"Bearer","expires_in":3599}`,
			mustContain: []string{`"access_token":"[REDACTED]"`, `"token_type":"Bearer"`, `"expires_in":3599`},
			mustExclude: []string{"token-abc123xyz"},
		},
		{
			name:        "JSON SecurityCredential in B2C payload",
			input:       `{"SecurityCredential":"encrypted-b2c-security-credential-string","CommandID":"SalaryPayment"}`,
			mustContain: []string{`"SecurityCredential":"[REDACTED]"`, `"CommandID":"SalaryPayment"`},
			mustExclude: []string{"encrypted-b2c-security-credential-string"},
		},
		{
			name:        "JSON numeric PIN field",
			input:       `{"pin":1234,"account":"SAV01"}`,
			mustContain: []string{`"pin":"[REDACTED]"`, `"account":"SAV01"`},
			mustExclude: []string{`"pin":1234`},
		},

		// Query strings and Form-encoded payloads
		{
			name: "URL query parameters with token and phone",
			input: "https://sandbox.safaricom.co.ke/mpesa/stkpushquery/v1/query?" +
				"token=my-secret-token&phone=254712345678&checkoutRequestID=ws_CO_0102",
			mustContain: []string{
				"token=[REDACTED]",
				"phone=2547***5678",
				"checkoutRequestID=ws_CO_0102",
			},
			mustExclude: []string{"my-secret-token", "254712345678"},
		},
		{
			name:        "Form-encoded credentials",
			input:       "password=super-secret-pw&consumer_secret=my-secret&client_id=my-app",
			mustContain: []string{"password=[REDACTED]", "consumer_secret=[REDACTED]", "client_id=my-app"},
			mustExclude: []string{"super-secret-pw", "my-secret"},
		},

		// XML / SOAP payloads
		{
			name: "SOAP XML credentials and phone",
			input: "<soapenv:Envelope><soapenv:Body><Password>secret-soap-password</Password>" +
				"<PhoneNumber>254712345678</PhoneNumber></soapenv:Body></soapenv:Envelope>",
			mustContain: []string{
				"<Password>[REDACTED]</Password>",
				"<PhoneNumber>2547***5678</PhoneNumber>",
			},
			mustExclude: []string{"secret-soap-password", "254712345678"},
		},
		{
			name:  "WS-Security SOAP XML credentials with attributes",
			input: `<wsse:Security><wsse:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText">raw-wsse-password-secret</wsse:Password></wsse:Security>`,
			mustContain: []string{
				`<wsse:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText">[REDACTED]</wsse:Password>`,
			},
			mustExclude: []string{"raw-wsse-password-secret"},
		},

		// Preservation of non-sensitive diagnostic identifiers
		{
			name:        "Preserve Safaricom Paybill shortcode 174379",
			input:       `"BusinessShortCode": "174379"`,
			mustContain: []string{`"BusinessShortCode": "174379"`},
			mustExclude: []string{`***`},
		},
		{
			name:        "Preserve timestamp YYYYMMDDHHMMSS",
			input:       `"Timestamp": "20261101235959"`,
			mustContain: []string{`"Timestamp": "20261101235959"`},
			mustExclude: []string{`***`},
		},
		{
			name:        "Preserve UUID transaction identifier",
			input:       `"transaction_id": "0192e212-70b1-7a5b-b999-72c1c696e1a1"`,
			mustContain: []string{`"transaction_id": "0192e212-70b1-7a5b-b999-72c1c696e1a1"`},
			mustExclude: []string{`***`},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := scrubber.Scrub(tt.input)

			for _, want := range tt.mustContain {
				assert.Contains(t, got, want, "Scrubbed output must contain expected diagnostic structure: %q\nFull output:\n%s", want, got)
			}
			for _, exclude := range tt.mustExclude {
				assert.NotContains(t, got, exclude, "Scrubbed output leaked sensitive data: %q\nFull output:\n%s", exclude, got)
			}
		})
	}
}

func TestZeroLeakage_Invariants(t *testing.T) {
	t.Parallel()

	scrubber := paykit.NewDefaultScrubber()

	secretCredentials := []string{
		"SECRET_DARAJA_PW_XYZ!",
		"SECRET_PESAPAL_CONSUMER_KEY_123",
		"SECRET_BEARER_TOKEN_ABCD9876",
		"SECRET_PASSKEY_999988887777",
		"SECRET_B2C_CREDENTIAL_BLOB",
	}

	rawPhoneNumbers := []string{
		"254712345678",
		"254799887766",
		"254111223344",
		"0712345678",
		"0112345678",
	}

	rawTranscript := fmt.Sprintf(`
POST /mpesa/stkpush/v1/processrequest HTTP/1.1
Host: sandbox.safaricom.co.ke
Authorization: Bearer %s
X-Api-Key: %s

{
  "BusinessShortCode": "174379",
  "Password": "%s",
  "Timestamp": "20261101235959",
  "TransactionType": "CustomerPayBillOnline",
  "Amount": 100,
  "PartyA": "%s",
  "PartyB": "174379",
  "PhoneNumber": "%s",
  "CallBackURL": "https://example.com/callback?token=%s&phone=%s",
  "AccountReference": "INV001",
  "passkey": "%s",
  "SecurityCredential": "%s",
  "secondary_phone": "%s",
  "local_phone": "%s",
  "airtel_local": "%s"
}
`,
		secretCredentials[2], // Bearer token
		secretCredentials[1], // API key
		secretCredentials[0], // Password
		rawPhoneNumbers[0],   // PartyA
		rawPhoneNumbers[0],   // PhoneNumber
		secretCredentials[2], // query token
		rawPhoneNumbers[1],   // query phone
		secretCredentials[3], // passkey
		secretCredentials[4], // SecurityCredential
		rawPhoneNumbers[2],   // secondary phone
		rawPhoneNumbers[3],   // local phone
		rawPhoneNumbers[4],   // airtel local
	)

	scrubbed := scrubber.Scrub(rawTranscript)

	// Verify ZERO leakage of sensitive credentials
	for _, secret := range secretCredentials {
		assert.NotContains(t, scrubbed, secret, "Invariant violation: credential leaked: %s", secret)
	}

	// Verify ZERO leakage of unmasked phone numbers
	for _, phone := range rawPhoneNumbers {
		assert.NotContains(t, scrubbed, phone, "Invariant violation: unmasked phone leaked: %s", phone)
	}

	// Verify diagnostic structures remain intact
	assert.Contains(t, scrubbed, "Authorization: Bearer [REDACTED]")
	assert.Contains(t, scrubbed, "X-Api-Key: [REDACTED]")
	assert.Contains(t, scrubbed, `"BusinessShortCode": "174379"`)
	assert.Contains(t, scrubbed, `"Password": "[REDACTED]"`)
	assert.Contains(t, scrubbed, `"PartyA": "2547***5678"`)
	assert.Contains(t, scrubbed, `"PhoneNumber": "2547***5678"`)
	assert.Contains(t, scrubbed, `"passkey": "[REDACTED]"`)
	assert.Contains(t, scrubbed, `"SecurityCredential": "[REDACTED]"`)
}

func TestNopScrubber(t *testing.T) {
	t.Parallel()

	nop := paykit.NopScrubber{}
	input := "Authorization: Bearer secret-token\nPhone: 254712345678\npassword=xyz"
	got := nop.Scrub(input)
	assert.Equal(t, input, got, "NopScrubber must return input unmodified")
}

func TestScrubbingHandler_LogSanitization(t *testing.T) {
	t.Parallel()

	t.Run("TextHandler sanitizes record message and attributes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		handler := paykit.NewScrubbingHandler(baseHandler, paykit.NewDefaultScrubber())
		logger := slog.New(handler)

		logger.Info(
			"Initiated charge for 254712345678 using Bearer super-secret-token",
			"password", "user-secret-pw",
			"consumer_secret", "pesapal-consumer-secret",
			"phone", "254712345678",
			"party_a", "254799887766",
			"pin", 1234,
			"status", "pending",
		)

		logOutput := buf.String()

		// Verify zero leakage
		assert.NotContains(t, logOutput, "super-secret-token")
		assert.NotContains(t, logOutput, "user-secret-pw")
		assert.NotContains(t, logOutput, "pesapal-consumer-secret")
		assert.NotContains(t, logOutput, "254712345678")
		assert.NotContains(t, logOutput, "254799887766")
		assert.NotContains(t, logOutput, "1234")

		// Verify expected masking
		assert.Contains(t, logOutput, "Initiated charge for 2547***5678 using Bearer [REDACTED]")
		assert.Contains(t, logOutput, "password=[REDACTED]")
		assert.Contains(t, logOutput, "consumer_secret=[REDACTED]")
		assert.Contains(t, logOutput, "phone=2547***5678")
		assert.Contains(t, logOutput, "party_a=2547***7766")
		assert.Contains(t, logOutput, "pin=[REDACTED]")
		assert.Contains(t, logOutput, "status=pending")
	})

	t.Run("JSONHandler sanitizes structured groups and pre-bound attributes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
		handler := paykit.NewScrubbingHandler(baseHandler, paykit.NewDefaultScrubber())

		// Test WithAttrs and WithGroup
		scopedLogger := slog.New(handler).
			With("caller_phone", "254712345678").
			WithGroup("auth")

		scopedLogger.Info(
			"User authentication attempt",
			"password", "secret-password-123",
			"token", "bearer-access-token",
		)

		logOutput := buf.String()

		// Verify zero leakage
		assert.NotContains(t, logOutput, "254712345678")
		assert.NotContains(t, logOutput, "secret-password-123")
		assert.NotContains(t, logOutput, "bearer-access-token")

		// Verify masked structures
		assert.Contains(t, logOutput, `"caller_phone":"2547***5678"`)
		assert.Contains(t, logOutput, `"password":"[REDACTED]"`)
		assert.Contains(t, logOutput, `"token":"[REDACTED]"`)
	})

	t.Run("NewScrubbingHandler prevents redundant wrapping", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		baseHandler := slog.NewTextHandler(&buf, nil)
		h1 := paykit.NewScrubbingHandler(baseHandler, nil)
		h2 := paykit.NewScrubbingHandler(h1, nil)

		assert.Same(t, h1, h2, "Wrapping an existing ScrubbingHandler must return the existing instance")
	})
}

func TestHTTPClient_Scrubber_DumpWriterAndLogger(t *testing.T) {
	t.Parallel()

	// Setup a mock payment gateway server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify raw request reaches server uncorrupted
		auth := r.Header.Get("Authorization")
		if auth != "Bearer raw-inbound-wire-token-999" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}

		// Ensure raw wire payload was NOT modified before network dispatch
		if !strings.Contains(string(body), "raw-inbound-password-secret") {
			http.Error(w, "corrupted body", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"raw-response-token-111","status":"00","PhoneNumber":"254712345678"}`))
	}))
	defer server.Close()

	var dumpBuf bytes.Buffer
	var logBuf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		DumpWriter: &dumpBuf,
		Logger:     logger,
	})
	client.SetTransport(server.Client().Transport)

	requestPayload := `{"Password":"raw-inbound-password-secret","PhoneNumber":"254712345678","PartyB":"174379","Amount":50}`
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+"/charge",
		strings.NewReader(requestPayload),
	)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer raw-inbound-wire-token-999")
	req.Header.Set("Idempotency-Key", "idem-key-001")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(context.Background(), req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify caller can read response body intact (unmodified by dump scrubbing)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(respBody), "raw-response-token-111")
	assert.Contains(t, string(respBody), "254712345678")

	// Verify wire dump output in DumpWriter
	dumpOutput := dumpBuf.String()

	// 1. Zero leakage in DumpWriter
	assert.NotContains(t, dumpOutput, "raw-inbound-wire-token-999", "DumpWriter leaked request Authorization token")
	assert.NotContains(t, dumpOutput, "raw-inbound-password-secret", "DumpWriter leaked request Password")
	assert.NotContains(t, dumpOutput, "raw-response-token-111", "DumpWriter leaked response token")
	assert.NotContains(t, dumpOutput, "254712345678", "DumpWriter leaked unmasked customer phone number")

	// 2. Expected masked fields and preserved diagnostic structures in DumpWriter
	assert.Contains(t, dumpOutput, "Authorization: Bearer [REDACTED]")
	assert.Contains(t, dumpOutput, `"Password":"[REDACTED]"`)
	assert.Contains(t, dumpOutput, `"PhoneNumber":"2547***5678"`)
	assert.Contains(t, dumpOutput, `"PartyB":"174379"`, "Diagnostic shortcode 174379 must remain intact")
	assert.Contains(t, dumpOutput, `"Amount":50`, "Diagnostic amount must remain intact")
	assert.Contains(t, dumpOutput, `"token":"[REDACTED]"`)

	// Verify debug logger output
	logOutput := logBuf.String()

	// 3. Zero leakage in Logger
	assert.NotContains(t, logOutput, "raw-inbound-wire-token-999", "Logger leaked Authorization token")
	assert.NotContains(t, logOutput, "raw-inbound-password-secret", "Logger leaked Password")
	assert.NotContains(t, logOutput, "254712345678", "Logger leaked phone number")

	// 4. Diagnostic logger events present
	assert.Contains(t, logOutput, "sending http request")
	assert.Contains(t, logOutput, "received http response")
	assert.Contains(t, logOutput, "status=200")
}

func TestHTTPClient_Scrubber_OptOutWithNopScrubber(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	var dumpBuf bytes.Buffer

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		DumpWriter: &dumpBuf,
		Scrubber:   paykit.NopScrubber{}, // Explicitly disable scrubbing
	})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+"/charge",
		strings.NewReader(`{"phone":"254712345678"}`),
	)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer unscrubbed-token")
	req.Header.Set("Idempotency-Key", "idem-key-nop")

	resp, err := client.Do(context.Background(), req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	dumpOutput := dumpBuf.String()
	assert.Contains(t, dumpOutput, "unscrubbed-token", "NopScrubber must leave tokens unscrubbed when explicitly configured")
	assert.Contains(t, dumpOutput, "254712345678", "NopScrubber must leave phone numbers unscrubbed when explicitly configured")
}
