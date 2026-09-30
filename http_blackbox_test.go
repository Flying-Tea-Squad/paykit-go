package paykit_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Flying-Tea-Squad/paykit-go"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestHTTPClient_IdempotencyHeader_Preserved(t *testing.T) {
	t.Parallel()

	var receivedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+"/charge",
		strings.NewReader(`{"amount":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected NewRequestWithContext error: %v", err)
	}
	req.Header.Set("Idempotency-Key", "custom-idempotency-key-12345")

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected client.Do error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got: %d", resp.StatusCode)
	}
	if receivedKey != "custom-idempotency-key-12345" {
		t.Errorf("expected Idempotency-Key 'custom-idempotency-key-12345', got %q", receivedKey)
	}
}

func TestHTTPClient_IdempotencyPolicy_Require_Default(t *testing.T) {
	t.Parallel()

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Default config should have IdempotencyPolicyRequire
	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	client.SetTransport(server.Client().Transport)

	mutatingMethods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range mutatingMethods {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequestWithContext(
				context.Background(),
				method,
				server.URL+"/resource",
				strings.NewReader(`{"test":true}`),
			)
			if err != nil {
				t.Fatalf("unexpected NewRequest error: %v", err)
			}

			_, err = client.Do(context.Background(), req)
			if !errors.Is(err, paykit.ErrMissingIdempotencyKeySentinel) {
				t.Errorf("expected ErrMissingIdempotencyKeySentinel for %s, got: %v", method, err)
			}
		})
	}

	if atomic.LoadInt32(&requestCount) != 0 {
		t.Errorf("expected 0 requests dispatched upstream, got: %d", requestCount)
	}
}

func TestHTTPClient_IdempotencyPolicy_AutoGenerate(t *testing.T) {
	t.Parallel()

	var receivedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		IdempotencyPolicy: paykit.IdempotencyPolicyAutoGenerate,
	})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+"/disburse",
		strings.NewReader(`{"amount":500}`),
	)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Do error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 Created, got: %d", resp.StatusCode)
	}
	if !uuidRegex.MatchString(receivedKey) {
		t.Errorf("expected generated key to match UUIDv4 format, got: %q", receivedKey)
	}
}

func TestHTTPClient_IdempotencyPolicy_Optional(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		// Return 500 to trigger retry evaluation
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		IdempotencyPolicy: paykit.IdempotencyPolicyOptional,
		RetryBaseDelay:    10 * time.Millisecond,
		MaxAttempts:       3,
	})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+"/charge",
		strings.NewReader(`{"amount":100}`),
	)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error on Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Under Optional policy, a mutating request lacking an Idempotency-Key MUST NOT be retried
	if totalAttempts := atomic.LoadInt32(&attempts); totalAttempts != 1 {
		t.Errorf("expected exactly 1 attempt for mutating request without idempotency key, got: %d", totalAttempts)
	}
}

func TestHTTPClient_NonMutatingMethods_NoKeyRequired(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		IdempotencyPolicy: paykit.IdempotencyPolicyRequire, // Even with Require
		RetryBaseDelay:    10 * time.Millisecond,
		MaxAttempts:       3,
	})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		server.URL+"/status/txn_123",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Do error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK after retries, got: %d", resp.StatusCode)
	}
	if totalAttempts := atomic.LoadInt32(&attempts); totalAttempts != 3 {
		t.Errorf("expected 3 attempts for idempotent GET, got: %d", totalAttempts)
	}
}

func TestHTTPClient_MutatingMethodWithKey_RetriesOn5xx(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		RetryBaseDelay: 10 * time.Millisecond,
		MaxAttempts:    3,
	})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		server.URL+"/charge",
		strings.NewReader(`{"amount":200}`),
	)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}
	req.Header.Set("Idempotency-Key", "key-retry-123")

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Do error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK after retries, got: %d", resp.StatusCode)
	}
	if totalAttempts := atomic.LoadInt32(&attempts); totalAttempts != 3 {
		t.Errorf("expected 3 attempts, got: %d", totalAttempts)
	}
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	client.SetTransport(server.Client().Transport)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before executing

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}

	_, err = client.Do(ctx, req)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestHTTPClient_429WithRetryAfter(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		MaxAttempts: 3,
	})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/rate-limited", nil)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected Do error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK after 429 retry, got: %d", resp.StatusCode)
	}
	if totalAttempts := atomic.LoadInt32(&attempts); totalAttempts != 2 {
		t.Errorf("expected 2 attempts, got: %d", totalAttempts)
	}
}

func TestHTTPClient_DoRequestConvenience(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	client.SetTransport(server.Client().Transport)

	req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
	if err != nil {
		t.Fatalf("unexpected NewRequest error: %v", err)
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		t.Fatalf("unexpected DoRequest error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got: %d", resp.StatusCode)
	}
}
