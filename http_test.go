package paykit

import (
	"bytes"
	"context"
	"crypto/tls"
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
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientConfig{RetryBaseDelay: time.Nanosecond})
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("amount=100"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
		w.Write([]byte("created"))
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
	defer resp.Body.Close()

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
