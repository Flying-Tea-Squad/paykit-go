package paykit

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
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

	// Default behavior: mutating methods are NEVER retryable, even with Idempotency-Key.
	tests := []struct {
		name      string
		method    string
		headerKey string
		retryable bool
	}{
		{"GET without key", http.MethodGet, "", true},
		{"GET with key", http.MethodGet, "key-123", true},
		{"HEAD without key", http.MethodHead, "", true},
		{"OPTIONS without key", http.MethodOptions, "", true},
		{"POST with key", http.MethodPost, "key-123", false},
		{"POST without key", http.MethodPost, "", false},
		{"PUT with key", http.MethodPut, "key-123", false},
		{"PUT without key", http.MethodPut, "", false},
		{"PATCH with key", http.MethodPatch, "key-123", false},
		{"PATCH without key", http.MethodPatch, "", false},
		{"DELETE with key", http.MethodDelete, "key-123", false},
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

func TestHTTPClient_IsRetryableMethod_OptIn(t *testing.T) {
	t.Parallel()

	client := NewHTTPClient(HTTPClientConfig{
		AllowMutatingRetries: true,
	})

	tests := []struct {
		name      string
		method    string
		headerKey string
		retryable bool
	}{
		{"GET without key", http.MethodGet, "", true},
		{"GET with key", http.MethodGet, "key-123", true},
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
			if got := client.isRetryableMethod(tt.method, tt.headerKey); got != tt.retryable {
				t.Errorf("client.isRetryableMethod(%q, %q) = %v, want %v", tt.method, tt.headerKey, got, tt.retryable)
			}
		})
	}
}

func TestHTTPClient_CanRetry(t *testing.T) {
	t.Parallel()

	defaultClient := NewHTTPClient(HTTPClientConfig{})
	optInClient := NewHTTPClient(HTTPClientConfig{AllowMutatingRetries: true})

	makeReq := func(method, key string) *http.Request {
		req, _ := http.NewRequest(method, "https://example.com/test", nil)
		if key != "" {
			req.Header.Set(headerIdempotencyKey, key)
		}
		return req
	}

	makeResp := func(status int) *http.Response {
		return &http.Response{StatusCode: status}
	}

	transientErr := errors.New("connection reset by peer")

	t.Run("nil request returns false", func(t *testing.T) {
		if defaultClient.canRetry(nil, makeResp(500), nil) {
			t.Error("expected false for nil request")
		}
		if canRetry(nil, makeResp(500), nil) {
			t.Error("expected false for nil request in package canRetry")
		}
	})

	t.Run("safe method GET retry evaluation", func(t *testing.T) {
		req := makeReq(http.MethodGet, "")
		if !defaultClient.canRetry(req, makeResp(500), nil) {
			t.Error("expected true for GET on 500")
		}
		if !defaultClient.canRetry(req, makeResp(429), nil) {
			t.Error("expected true for GET on 429")
		}
		if !defaultClient.canRetry(req, nil, transientErr) {
			t.Error("expected true for GET on transient error")
		}
		if defaultClient.canRetry(req, makeResp(400), nil) {
			t.Error("expected false for GET on 400")
		}
		if defaultClient.canRetry(req, makeResp(200), nil) {
			t.Error("expected false for GET on 200")
		}
		if defaultClient.canRetry(req, nil, context.Canceled) {
			t.Error("expected false for GET on context.Canceled")
		}
		if defaultClient.canRetry(req, nil, nil) {
			t.Error("expected false when both resp and err are nil")
		}
	})

	t.Run("mutating method POST default client blocks retries even with key", func(t *testing.T) {
		req := makeReq(http.MethodPost, "idem-key-1")
		if defaultClient.canRetry(req, makeResp(500), nil) {
			t.Error("expected false for POST with key on default client under 500")
		}
		if defaultClient.canRetry(req, makeResp(429), nil) {
			t.Error("expected false for POST with key on default client under 429")
		}
		if defaultClient.canRetry(req, nil, transientErr) {
			t.Error("expected false for POST with key on default client under transient error")
		}
	})

	t.Run("mutating method POST opt-in client permits retries only with key", func(t *testing.T) {
		reqWithKey := makeReq(http.MethodPost, "idem-key-1")
		reqWithoutKey := makeReq(http.MethodPost, "")

		if !optInClient.canRetry(reqWithKey, makeResp(500), nil) {
			t.Error("expected true for POST with key on opt-in client under 500")
		}
		if !optInClient.canRetry(reqWithKey, makeResp(429), nil) {
			t.Error("expected true for POST with key on opt-in client under 429")
		}
		if !optInClient.canRetry(reqWithKey, nil, transientErr) {
			t.Error("expected true for POST with key on opt-in client under transient error")
		}

		if optInClient.canRetry(reqWithoutKey, makeResp(500), nil) {
			t.Error("expected false for POST without key on opt-in client under 500")
		}
		if optInClient.canRetry(reqWithoutKey, nil, transientErr) {
			t.Error("expected false for POST without key on opt-in client under transient error")
		}
	})
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

func TestDefaultMaxRetryAfterDelay(t *testing.T) {
	t.Parallel()

	if defaultMaxRetryAfterDelay != 60*time.Second {
		t.Errorf("expected defaultMaxRetryAfterDelay to be 60s, got: %v", defaultMaxRetryAfterDelay)
	}
}

func TestCalculateBackoffWithJitter(t *testing.T) {
	t.Parallel()

	baseDelay := 100 * time.Millisecond

	if got := calculateBackoffWithJitter(0, 1); got != 0 {
		t.Errorf("expected 0 for zero base delay, got: %v", got)
	}
	if got := calculateBackoffWithJitter(-time.Second, 1); got != 0 {
		t.Errorf("expected 0 for negative base delay, got: %v", got)
	}

	for attempt := 1; attempt <= 4; attempt++ {
		base := baseDelay * time.Duration(1<<uint(attempt-1))
		minBound := base * 80 / 100
		maxBound := base * 120 / 100

		seenDifferent := false
		var firstDelay time.Duration

		for i := 0; i < 50; i++ {
			got := calculateBackoffWithJitter(baseDelay, attempt)
			if got < minBound || got > maxBound {
				t.Fatalf("attempt %d iteration %d: delay %v outside expected [%v, %v]",
					attempt, i, got, minBound, maxBound)
			}
			if i == 0 {
				firstDelay = got
			} else if got != firstDelay {
				seenDifferent = true
			}
		}

		if !seenDifferent {
			t.Errorf("attempt %d: expected randomized jitter variation across 50 iterations, but all were %v",
				attempt, firstDelay)
		}
	}
}

func TestHTTPClientConcurrentZeroValueClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HTTPClient{}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Error(err)
				return
			}
			resp, err := client.Do(context.Background(), req)
			if err != nil {
				t.Error(err)
				return
			}
			_ = resp.Body.Close()
		}()
	}
	close(start)
	wg.Wait()
}
