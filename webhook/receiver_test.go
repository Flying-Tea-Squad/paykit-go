package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGatewayWebhookHandler implements paykit.WebhookHandler for testing.
type mockGatewayWebhookHandler struct {
	provider string
	verifyFn func(r *http.Request) (*paykit.Event, error)
}

func (m *mockGatewayWebhookHandler) ParseAndVerify(r *http.Request) (*paykit.Event, error) {
	if m.verifyFn != nil {
		return m.verifyFn(r)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return &paykit.Event{
		ID:            "evt_001",
		Provider:      m.provider,
		Type:          "charge.completed",
		TransactionID: "txn_test_123",
		Status:        paykit.StatusSuccess,
		Amount:        paykit.NewKES(1500),
		Phone:         "254712345678",
		Timestamp:     time.Date(2026, 10, 27, 22, 0, 0, 0, time.UTC),
		Raw:           json.RawMessage(body),
	}, nil
}

// Ensure mockGatewayWebhookHandler satisfies paykit.WebhookHandler at compile time.
var _ paykit.WebhookHandler = (*mockGatewayWebhookHandler)(nil)

func TestWebhookReceiver_TableDrivenScenarios(t *testing.T) {
	tests := []struct {
		name               string
		providerRoute      string
		method             string
		requestPath        string
		requestHeaders     map[string]string
		requestBody        string
		handlerError       error
		appHandlerError    error
		expectedStatusCode int
		expectedHeader     map[string]string
		expectedJSONBody   string
		validateEvent      func(t *testing.T, evt *paykit.Event)
	}{
		{
			name:               "Mpesa callback success with Safaricom Daraja acknowledgment",
			providerRoute:      "mpesa",
			method:             http.MethodPost,
			requestPath:        "/webhooks/mpesa",
			requestBody:        `{"Body":{"stkCallback":{"ResultCode":0,"ResultDesc":"Success"}}}`,
			expectedStatusCode: http.StatusOK,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
			expectedJSONBody:   `{"ResultCode":0,"ResultDesc":"Accepted"}`,
			validateEvent: func(t *testing.T, evt *paykit.Event) {
				assert.Equal(t, "mpesa", evt.Provider)
				assert.Equal(t, "charge.completed", evt.Type)
				assert.Equal(t, "txn_test_123", evt.TransactionID)
				assert.Equal(t, paykit.StatusSuccess, evt.Status)
				assert.Equal(t, paykit.NewKES(1500), evt.Amount)
				assert.Equal(t, "254712345678", evt.Phone)
			},
		},
		{
			name:               "Airtel callback success with Airtel acknowledgment",
			providerRoute:      "airtel",
			method:             http.MethodPost,
			requestPath:        "/webhooks/airtel",
			requestBody:        `{"transaction":{"id":"airtel_001","status":"success"}}`,
			expectedStatusCode: http.StatusOK,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
			expectedJSONBody:   `{"status":"success","message":"Accepted"}`,
			validateEvent: func(t *testing.T, evt *paykit.Event) {
				assert.Equal(t, "airtel", evt.Provider)
			},
		},
		{
			name:               "Pesapal IPN callback success with Pesapal acknowledgment",
			providerRoute:      "pesapal",
			method:             http.MethodPost,
			requestPath:        "/webhooks/pesapal",
			requestBody:        `{"OrderNotificationType":"IPNCHANGE","OrderTrackingId":"pesapal_trk_123"}`,
			expectedStatusCode: http.StatusOK,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
			expectedJSONBody:   `{"status":"200","message":"OK"}`,
			validateEvent: func(t *testing.T, evt *paykit.Event) {
				assert.Equal(t, "pesapal", evt.Provider)
			},
		},
		{
			name:               "Generic payment provider fallback acknowledgment",
			providerRoute:      "custompay",
			method:             http.MethodPost,
			requestPath:        "/webhooks/custompay",
			requestBody:        `{"event":"charge"}`,
			expectedStatusCode: http.StatusOK,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
			expectedJSONBody:   `{"status":"success","message":"Webhook processed successfully"}`,
		},
		{
			name:               "Invalid cryptographic signature rejected with HTTP 400",
			providerRoute:      "mpesa",
			method:             http.MethodPost,
			requestPath:        "/webhooks/mpesa",
			requestBody:        `{"Body":{"stkCallback":{"ResultCode":0}}}`,
			handlerError:       errors.New("invalid signature: HMAC verification failed"),
			expectedStatusCode: http.StatusBadRequest,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
		},
		{
			name:               "Malformed JSON payload rejected with HTTP 400",
			providerRoute:      "mpesa",
			method:             http.MethodPost,
			requestPath:        "/webhooks/mpesa",
			requestBody:        `{invalid-json`,
			handlerError:       errors.New("unexpected EOF"),
			expectedStatusCode: http.StatusBadRequest,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
		},
		{
			name:               "Route mismatch / unregistered provider returns HTTP 404",
			providerRoute:      "mpesa",
			method:             http.MethodPost,
			requestPath:        "/webhooks/unknown-provider",
			requestBody:        `{}`,
			expectedStatusCode: http.StatusNotFound,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
		},
		{
			name:               "HTTP GET disallowed with HTTP 405 Method Not Allowed",
			providerRoute:      "mpesa",
			method:             http.MethodGet,
			requestPath:        "/webhooks/mpesa",
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedHeader: map[string]string{
				"Allow":        "POST",
				"Content-Type": "application/json",
			},
		},
		{
			name:               "HTTP PUT disallowed with HTTP 405 Method Not Allowed",
			providerRoute:      "mpesa",
			method:             http.MethodPut,
			requestPath:        "/webhooks/mpesa",
			requestBody:        `{}`,
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedHeader: map[string]string{
				"Allow":        "POST",
				"Content-Type": "application/json",
			},
		},
		{
			name:               "Application event handler error returns HTTP 500",
			providerRoute:      "mpesa",
			method:             http.MethodPost,
			requestPath:        "/webhooks/mpesa",
			requestBody:        `{"test":true}`,
			appHandlerError:    errors.New("database transaction conflict"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedHeader:     map[string]string{"Content-Type": "application/json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedEvent *paykit.Event

			gwHandler := &mockGatewayWebhookHandler{
				provider: tt.providerRoute,
				verifyFn: func(r *http.Request) (*paykit.Event, error) {
					if tt.handlerError != nil {
						return nil, tt.handlerError
					}
					body, _ := io.ReadAll(r.Body)
					return &paykit.Event{
						ID:            "evt_001",
						Provider:      tt.providerRoute,
						Type:          "charge.completed",
						TransactionID: "txn_test_123",
						Status:        paykit.StatusSuccess,
						Amount:        paykit.NewKES(1500),
						Phone:         "254712345678",
						Timestamp:     time.Date(2026, 10, 27, 22, 0, 0, 0, time.UTC),
						Raw:           json.RawMessage(body),
					}, nil
				},
			}

			routes := map[string]paykit.WebhookHandler{
				tt.providerRoute: gwHandler,
			}

			eventHandler := func(_ context.Context, evt *paykit.Event) error {
				capturedEvent = evt
				return tt.appHandlerError
			}

			rcv := webhook.NewReceiver(routes, eventHandler)

			server := httptest.NewServer(rcv)
			defer server.Close()

			var reqBody io.Reader
			if tt.requestBody != "" {
				reqBody = bytes.NewBufferString(tt.requestBody)
			}

			req, err := http.NewRequest(tt.method, server.URL+tt.requestPath, reqBody)
			require.NoError(t, err)

			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			client := server.Client()
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			for k, v := range tt.expectedHeader {
				assert.Equal(t, v, resp.Header.Get(k))
			}

			if tt.expectedJSONBody != "" {
				bodyBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectedJSONBody, string(bodyBytes))
			}

			if tt.validateEvent != nil && capturedEvent != nil {
				tt.validateEvent(t, capturedEvent)
			}
		})
	}
}

func TestWebhookReceiver_StandardServeMuxIntegration(t *testing.T) {
	var dispatchedEvent *paykit.Event

	routes := map[string]paykit.WebhookHandler{
		"mpesa": &mockGatewayWebhookHandler{provider: "mpesa"},
	}

	rcv := webhook.NewReceiver(routes, func(_ context.Context, evt *paykit.Event) error {
		dispatchedEvent = evt
		return nil
	})

	// Mount on net/http standard ServeMux
	mux := http.NewServeMux()
	mux.Handle("/api/v1/webhooks/", http.StripPrefix("/api/v1", rcv))

	server := httptest.NewServer(mux)
	defer server.Close()

	payload := `{"ResultCode":0,"Amount":100}`
	resp, err := http.Post(server.URL+"/api/v1/webhooks/mpesa", "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, dispatchedEvent)
	assert.Equal(t, "mpesa", dispatchedEvent.Provider)
}

func TestWebhookReceiver_FrameworkAdapterPatterns(t *testing.T) {
	// Verifies compatibility with standard HTTP handler wrapping patterns:
	// - Gin: gin.WrapH(receiver)
	// - Echo: echo.WrapHandler(receiver)
	// - Chi: r.Mount("/webhooks", receiver)
	routes := map[string]paykit.WebhookHandler{
		"airtel": &mockGatewayWebhookHandler{provider: "airtel"},
	}

	var called bool
	rcv := webhook.NewReceiver(routes, func(_ context.Context, _ *paykit.Event) error {
		called = true
		return nil
	})

	// Universal http.Handler interface satisfaction
	var handler http.Handler = rcv

	// Chi-compatible adapter pattern (Mount on sub-router)
	chiAdapter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/airtel", bytes.NewBufferString(`{}`))

	chiAdapter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}
