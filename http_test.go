package paykit

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientRetriesAndReplaysRequestBody(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if string(body) != "amount=100" {
			t.Fatalf("unexpected request body: %q", body)
		}

		if attempts.Load() < 3 {
			http.Error(w, "try again", http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("amount=100"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	// POST is non-idempotent; the Idempotency-Key header opts in to retry.
	req.Header.Set("Idempotency-Key", "test-idem-key-001")

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestHTTPClientCapsRetryAttemptsAtThree(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "still unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientConfig{
		MaxAttempts:    10,
		RetryBaseDelay: time.Nanosecond,
	})
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected final 503, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected retry cap of 3 attempts, got %d", attempts.Load())
	}
}

func TestHTTPClientDumpWriterAndLogger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer server.Close()

	var dump bytes.Buffer
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewHTTPClient(HTTPClientConfig{
		DumpWriter: &dump,
		Logger:     logger,
	})

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != "created" {
		t.Fatalf("expected response body to remain readable, got %q", body)
	}

	dumpText := dump.String()
	for _, want := range []string{"POST ", "payload", "HTTP/1.1 201 Created", "X-Test: yes", "created"} {
		if !strings.Contains(dumpText, want) {
			t.Fatalf("dump missing %q:\n%s", want, dumpText)
		}
	}

	logText := logs.String()
	for _, want := range []string{"sending http request", "received http response", "status=201"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("logs missing %q:\n%s", want, logText)
		}
	}
}

func TestNewHTTPClientAppliesConfig(t *testing.T) {
	tlsConfig := &tls.Config{ServerName: "payments.example.test"}
	client := NewHTTPClient(HTTPClientConfig{
		ConnectionTimeout: 2 * time.Second,
		RequestTimeout:    5 * time.Second,
		TLSConfig:         tlsConfig,
		MaxAttempts:       2,
		RetryBaseDelay:    time.Millisecond,
	})

	if client.client.Timeout != 5*time.Second {
		t.Fatalf("expected request timeout 5s, got %s", client.client.Timeout)
	}
	if client.maxAttempts != 2 {
		t.Fatalf("expected max attempts 2, got %d", client.maxAttempts)
	}
	if client.retryBaseDelay != time.Millisecond {
		t.Fatalf("expected retry base delay 1ms, got %s", client.retryBaseDelay)
	}

	transport, ok := client.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.client.Transport)
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLS config to be set")
	}
	if transport.TLSClientConfig == tlsConfig {
		t.Fatal("expected TLS config to be cloned")
	}
	if transport.TLSClientConfig.ServerName != tlsConfig.ServerName {
		t.Fatalf("expected server name %q, got %q", tlsConfig.ServerName, transport.TLSClientConfig.ServerName)
	}
}

// TestHTTPClientNonIdempotentMethodNotRetried verifies that POST and PATCH
// requests are sent exactly once when no Idempotency-Key header is present,
// even when the server returns a retryable 5xx status code.
func TestHTTPClientNonIdempotentMethodNotRetried(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		method := method
		t.Run(method, func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			}))
			defer server.Close()

			client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})
			req, err := http.NewRequest(method, server.URL, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			// No Idempotency-Key header — must not retry.

			resp, err := client.Do(context.Background(), req)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", method, err)
			}
			_ = resp.Body.Close()

			if attempts.Load() != 1 {
				t.Fatalf("%s: expected exactly 1 attempt without Idempotency-Key, got %d", method, attempts.Load())
			}
		})
	}
}

// TestHTTPClientGetBodyRequiredForRetry verifies that a retryable POST with a
// raw io.ReadCloser body (GetBody == nil) is rejected with a clear error before
// any request is sent.
func TestHTTPClientGetBodyRequiredForRetry(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})

	// io.NopCloser wraps an arbitrary reader; net/http does NOT set GetBody for it.
	req, err := http.NewRequest(http.MethodPost, server.URL, io.NopCloser(strings.NewReader("data")))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Idempotency-Key", "idem-001") // opts POST in to retry path

	_, err = client.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected an error for missing GetBody, got nil")
	}
	if !strings.Contains(err.Error(), "GetBody") {
		t.Fatalf("expected GetBody mention in error, got: %v", err)
	}
	if attempts.Load() != 0 {
		t.Fatalf("expected 0 server attempts, got %d", attempts.Load())
	}
}

// TestHTTPClient429WithRetryAfterRetries verifies that a 429 response carrying
// a Retry-After header causes the client to sleep and retry.
func TestHTTPClient429WithRetryAfterRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0") // zero-second delay
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success after 429 retries, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

// TestHTTPClient429WithoutRetryAfterErrors verifies that a 429 with no
// Retry-After header results in an immediate error rather than a blind retry.
func TestHTTPClient429WithoutRetryAfterErrors(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		// Intentionally omit Retry-After.
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	_, err = client.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for 429 without Retry-After, got nil")
	}
	if !strings.Contains(err.Error(), "no Retry-After") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", attempts.Load())
	}
}

// TestHTTPClientContextCancelledNotRetried verifies that a pre-cancelled context
// causes the client to return context.Canceled without making any server attempts.
func TestHTTPClientContextCancelledNotRetried(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "should not reach", http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel before any request

	client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	_, err = client.Do(ctx, req)
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if attempts.Load() != 0 {
		t.Fatalf("expected 0 server attempts with canceled context, got %d", attempts.Load())
	}
}
