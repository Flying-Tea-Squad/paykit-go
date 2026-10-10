package paykit

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"testing"
	"time"
)

var uuidFormatRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestGenerateIdempotencyKey(t *testing.T) {
	t.Parallel()

	const iterations = 5000
	generated := make(map[string]struct{}, iterations)

	for i := 0; i < iterations; i++ {
		key, err := generateIdempotencyKey()
		if err != nil {
			t.Fatalf("unexpected error generating key: %v", err)
		}

		if len(key) != 36 {
			t.Fatalf("expected key length 36, got %d: %q", len(key), key)
		}

		if !uuidFormatRegex.MatchString(key) {
			t.Fatalf("generated key does not conform to UUIDv4: %q", key)
		}

		if _, exists := generated[key]; exists {
			t.Fatalf("duplicate key generated at iteration %d: %q", i, key)
		}
		generated[key] = struct{}{}
	}
}

func TestIsMutatingMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method   string
		mutating bool
	}{
		{http.MethodPost, true},
		{http.MethodPut, true},
		{http.MethodPatch, true},
		{http.MethodDelete, true},
		{http.MethodGet, false},
		{http.MethodHead, false},
		{http.MethodOptions, false},
		{"TRACE", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isMutatingMethod(tt.method); got != tt.mutating {
				t.Errorf("isMutatingMethod(%q) = %v, want %v", tt.method, got, tt.mutating)
			}
		})
	}
}

func TestIsRetryableMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		method    string
		headerKey string
		retryable bool
	}{
		{"GET without key", http.MethodGet, "", true},
		{"POST with key", http.MethodPost, "key-123", true},
		{"POST without key", http.MethodPost, "", false},
		{"PUT with key", http.MethodPut, "key-123", true},
		{"PUT without key", http.MethodPut, "", false},
		{"PATCH with key", http.MethodPatch, "key-123", true},
		{"PATCH without key", http.MethodPatch, "", false},
		{"DELETE with key", http.MethodDelete, "key-123", true},
		{"DELETE without key", http.MethodDelete, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "https://example.com", nil)
			if tt.headerKey != "" {
				req.Header.Set(headerIdempotencyKey, tt.headerKey)
			}

			if got := isRetryableMethod(req); got != tt.retryable {
				t.Errorf("isRetryableMethod() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestDumpWriter_RaceSafety(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	client := &HTTPClient{
		dumpWriter: &buf,
	}

	var wg sync.WaitGroup
	goroutines := 50

	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "https://example.com/test", nil)
			client.dumpRequest(req)
		}()
		go func() {
			defer wg.Done()
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}
			client.dumpResponse(resp)
		}()
	}

	wg.Wait()

	if buf.Len() == 0 {
		t.Error("expected dump buffer to contain written data")
	}
}

func TestShouldRetryError(t *testing.T) {
	t.Parallel()

	if shouldRetryError(nil) {
		t.Error("expected false for nil error")
	}
	if shouldRetryError(context.Canceled) {
		t.Error("expected false for context.Canceled")
	}
	if shouldRetryError(context.DeadlineExceeded) {
		t.Error("expected false for context.DeadlineExceeded")
	}

	certErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: x509.CertificateInvalidError{},
	}
	if shouldRetryError(certErr) {
		t.Error("expected false for certificate invalid error")
	}

	dnsErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: &net.DNSError{IsNotFound: true},
	}
	if shouldRetryError(dnsErr) {
		t.Error("expected false for permanent DNS error")
	}

	generalErr := errors.New("connection reset by peer")
	if !shouldRetryError(generalErr) {
		t.Error("expected true for transient error")
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	// Seconds format
	d, ok := parseRetryAfter("120")
	if !ok || d != 120*time.Second {
		t.Errorf("expected 120s, got %v (ok=%v)", d, ok)
	}

	// Empty
	_, ok = parseRetryAfter("")
	if ok {
		t.Error("expected ok=false for empty string")
	}

	// Invalid
	_, ok = parseRetryAfter("invalid-date")
	if ok {
		t.Error("expected ok=false for invalid string")
	}

	// Future HTTP-date
	future := time.Now().Add(60 * time.Second).UTC().Format(http.TimeFormat)
	d, ok = parseRetryAfter(future)
	if !ok || d <= 0 || d > 65*time.Second {
		t.Errorf("expected ~60s, got %v (ok=%v)", d, ok)
	}
}
