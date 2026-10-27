package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyHandler struct {
	parseFunc func(r *http.Request) (*paykit.Event, error)
}

func (d *dummyHandler) ParseAndVerify(r *http.Request) (*paykit.Event, error) {
	if d.parseFunc != nil {
		return d.parseFunc(r)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &paykit.Event{
		ID:            "evt_dummy",
		Provider:      "dummy",
		Type:          "test.event",
		TransactionID: "txn_123",
		Raw:           json.RawMessage(body),
	}, nil
}

type dummyAcknowledgerHandler struct {
	dummyHandler
	ackCalled bool
}

func (d *dummyAcknowledgerHandler) Acknowledge(w http.ResponseWriter, _ *http.Request, _ *paykit.Event) error {
	d.ackCalled = true
	w.Header().Set("X-Custom-Ack", "true")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"custom":true}`))
	return nil
}

func TestNewReceiver_DefensiveCopyAndNormalization(t *testing.T) {
	routes := map[string]paykit.WebhookHandler{
		"Mpesa ":   &dummyHandler{},
		" AIRTEL ": &dummyHandler{},
		"":         &dummyHandler{}, // empty key should be skipped
		"nil":      nil,             // nil handler should be skipped
	}

	rcv := NewReceiver(routes, nil)
	require.NotNil(t, rcv)

	// Modifying original map should not affect internal state
	routes["new_provider"] = &dummyHandler{}
	_, ok := rcv.Handler("new_provider")
	assert.False(t, ok)

	// Case insensitive normalized lookups
	_, ok = rcv.Handler("mpesa")
	assert.True(t, ok)
	_, ok = rcv.Handler("MPESA")
	assert.True(t, ok)
	_, ok = rcv.Handler("airtel")
	assert.True(t, ok)
	_, ok = rcv.Handler("")
	assert.False(t, ok)
	_, ok = rcv.Handler("nil")
	assert.False(t, ok)
}

func TestReceiver_RegisterAndHandler(t *testing.T) {
	rcv := NewReceiver(nil, nil)

	// Register with mixed casing and spaces
	rcv.Register(" Pesapal ", &dummyHandler{})
	h, ok := rcv.Handler("pesapal")
	assert.True(t, ok)
	assert.NotNil(t, h)

	// Overwrite handler
	newHandler := &dummyHandler{}
	rcv.Register("pesapal", newHandler)
	h2, ok := rcv.Handler("PESAPAL")
	assert.True(t, ok)
	assert.Same(t, newHandler, h2)

	// Ignored registrations
	rcv.Register("", &dummyHandler{})
	_, ok = rcv.Handler("")
	assert.False(t, ok)

	rcv.Register("invalid_nil", nil)
	_, ok = rcv.Handler("invalid_nil")
	assert.False(t, ok)
}

func TestReceiver_ResolveProvider_PathNormalization(t *testing.T) {
	rcv := NewReceiver(map[string]paykit.WebhookHandler{
		"mpesa":  &dummyHandler{},
		"airtel": &dummyHandler{},
	}, nil)

	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{
			name:     "standard path",
			path:     "/webhooks/mpesa",
			expected: "mpesa",
		},
		{
			name:     "trailing slash",
			path:     "/webhooks/mpesa/",
			expected: "mpesa",
		},
		{
			name:     "uppercase path",
			path:     "/webhooks/MPESA",
			expected: "mpesa",
		},
		{
			name:     "direct path without prefix",
			path:     "/mpesa",
			expected: "mpesa",
		},
		{
			name:     "nested subpath matching provider segment",
			path:     "/api/v1/callbacks/airtel/stk",
			expected: "airtel",
		},
		{
			name:     "configured path prefix",
			path:     "/api/v1/webhooks/mpesa",
			prefix:   "/api/v1/webhooks",
			expected: "mpesa",
		},
		{
			name:     "configured path prefix with trailing slash",
			path:     "/api/v1/webhooks/airtel/",
			prefix:   "/api/v1/webhooks/",
			expected: "airtel",
		},
		{
			name:     "empty root path",
			path:     "/",
			expected: "",
		},
		{
			name:     "unknown provider",
			path:     "/webhooks/unknown",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.prefix != "" {
				rcv.pathPrefix = tt.prefix
				defer func() { rcv.pathPrefix = "" }()
			}
			resolved := rcv.resolveProvider(r)
			assert.Equal(t, tt.expected, resolved)
		})
	}
}

func TestReceiver_CustomRouteMatcher(t *testing.T) {
	customMatcher := func(r *http.Request) string {
		return r.Header.Get("X-Payment-Provider")
	}

	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": &dummyHandler{}},
		nil,
		WithRouteMatcher(customMatcher),
	)

	req := httptest.NewRequest(http.MethodPost, "/any/arbitrary/path", nil)
	req.Header.Set("X-Payment-Provider", "mpesa")

	resolved := rcv.resolveProvider(req)
	assert.Equal(t, "mpesa", resolved)
}

func TestReceiver_BodyLimitAndReplay(t *testing.T) {
	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": &dummyHandler{}},
		nil,
		WithMaxBodyBytes(100),
	)

	// Payload within limit
	validPayload := bytes.Repeat([]byte("a"), 50)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader(validPayload))
	buffered, err := rcv.readAndBufferPayload(req)
	require.NoError(t, err)
	assert.Equal(t, validPayload, buffered)

	// Verify r.Body and r.GetBody can be read repeatedly
	firstRead, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, validPayload, firstRead)

	replayedReader, err := req.GetBody()
	require.NoError(t, err)
	secondRead, err := io.ReadAll(replayedReader)
	require.NoError(t, err)
	assert.Equal(t, validPayload, secondRead)

	// Payload exceeding limit
	oversizedPayload := bytes.Repeat([]byte("b"), 101)
	reqOver := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader(oversizedPayload))
	_, err = rcv.readAndBufferPayload(reqOver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum limit")

	// Nil body check
	reqNil := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", nil)
	reqNil.Body = nil
	_, err = rcv.readAndBufferPayload(reqNil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request body is nil")
}

func TestReceiver_WithMaxBodyBytes_ZeroOrNegativeFallback(t *testing.T) {
	rcv := NewReceiver(nil, nil, WithMaxBodyBytes(0))
	assert.Equal(t, int64(DefaultMaxBodyBytes), rcv.maxBodyBytes)

	rcvNegative := NewReceiver(nil, nil, WithMaxBodyBytes(-50))
	assert.Equal(t, int64(DefaultMaxBodyBytes), rcvNegative.maxBodyBytes)
}

func TestReceiver_HandlerAcknowledgerTakesPrecedence(t *testing.T) {
	ackHandler := &dummyAcknowledgerHandler{}
	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": ackHandler},
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader([]byte(`{"test":true}`)))
	w := httptest.NewRecorder()

	rcv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.True(t, ackHandler.ackCalled)
	assert.Equal(t, "true", w.Header().Get("X-Custom-Ack"))
	assert.JSONEq(t, `{"custom":true}`, w.Body.String())
}

func TestReceiver_WithCustomAcknowledgerOption(t *testing.T) {
	customAckCalled := false
	customAck := AcknowledgeFunc(func(w http.ResponseWriter, _ *http.Request, _ *paykit.Event) error {
		customAckCalled = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"override":true}`))
		return nil
	})

	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": &dummyHandler{}},
		nil,
		WithAcknowledger("mpesa", customAck),
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader([]byte(`{"hello":"world"}`)))
	w := httptest.NewRecorder()

	rcv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, customAckCalled)
	assert.JSONEq(t, `{"override":true}`, w.Body.String())
}

func TestReceiver_NilEventHandler_SucceedsWithAck(t *testing.T) {
	// If eventHandler is nil, the receiver should still verify and acknowledge successfully
	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": &dummyHandler{}},
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	rcv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"ResultCode":0,"ResultDesc":"Accepted"}`, w.Body.String())
}

func TestReceiver_NilEventFromHandler_Returns400(t *testing.T) {
	nilEventHandler := &dummyHandler{
		parseFunc: func(_ *http.Request) (*paykit.Event, error) {
			return nil, nil // nil event with nil error
		},
	}

	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": nilEventHandler},
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	rcv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "empty event")
}

func TestReceiver_Concurrency_RegisterAndServeHTTP(t *testing.T) {
	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": &dummyHandler{}},
		func(_ context.Context, _ *paykit.Event) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		},
	)

	var wg sync.WaitGroup
	workers := 10
	iterations := 20

	// Concurrent callers sending ServeHTTP
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader([]byte(`{"test":1}`)))
				w := httptest.NewRecorder()
				rcv.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("expected 200 OK, got %d", w.Code)
				}
			}
		}()
	}

	// Concurrent writer registering new routes
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				rcv.Register("airtel", &dummyHandler{})
				h, ok := rcv.Handler("airtel")
				if !ok || h == nil {
					t.Errorf("expected airtel to be registered")
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestReceiver_StructuredLogging(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	rcv := NewReceiver(
		map[string]paykit.WebhookHandler{"mpesa": &dummyHandler{}},
		func(_ context.Context, _ *paykit.Event) error {
			return errors.New("database connection down")
		},
		WithLogger(logger),
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/mpesa", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()

	rcv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	logs := logBuffer.String()
	assert.Contains(t, logs, "database connection down")
	assert.Contains(t, logs, "application event handler error")
}
