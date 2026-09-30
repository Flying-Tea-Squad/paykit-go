package paykit

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// IdempotencyPolicy controls how HTTPClient handles missing Idempotency-Key headers
// on mutating HTTP requests (POST, PUT, PATCH, DELETE).
type IdempotencyPolicy int

const (
	// IdempotencyPolicyRequire mandates that mutating requests provide an Idempotency-Key header;
	// requests lacking a key return ErrMissingIdempotencyKeySentinel immediately without sending.
	// This is the default policy for financial transaction safety.
	IdempotencyPolicyRequire IdempotencyPolicy = iota

	// IdempotencyPolicyAutoGenerate automatically generates a cryptographically secure UUIDv4
	// for mutating requests if no Idempotency-Key header was supplied.
	IdempotencyPolicyAutoGenerate

	// IdempotencyPolicyOptional allows mutating requests to proceed without an Idempotency-Key header,
	// but disables automatic retries for that request to prevent duplicate operations.
	IdempotencyPolicyOptional
)

const (
	defaultConnectionTimeout = 10 * time.Second
	defaultRequestTimeout    = 30 * time.Second
	defaultRetryBaseDelay    = 100 * time.Millisecond
	defaultMaxAttempts       = 3
	headerIdempotencyKey     = "Idempotency-Key"
)

// HTTPClientConfig configures the shared transport client.
type HTTPClientConfig struct {
	// ConnectionTimeout specifies the maximum duration waiting for a network dial to complete.
	ConnectionTimeout time.Duration

	// RequestTimeout specifies a time limit for requests made by this client.
	RequestTimeout time.Duration

	// TLSConfig contains custom TLS client configuration. It is cloned to prevent concurrent mutations.
	TLSConfig *tls.Config

	// Logger is an optional structured logger. When provided, debug details of request/response attempts are recorded.
	Logger *slog.Logger

	// DumpWriter is an optional writer to which full wire dumps of requests and responses will be written.
	DumpWriter io.Writer

	// RetryBaseDelay is the initial backoff delay between retry attempts.
	RetryBaseDelay time.Duration

	// MaxAttempts is the maximum number of attempts (initial attempt + retries) for retryable operations.
	// Values greater than 3 are clamped to 3.
	MaxAttempts int

	// IdempotencyPolicy specifies the handling strategy for mutating requests missing an Idempotency-Key header.
	// Defaults to IdempotencyPolicyRequire.
	IdempotencyPolicy IdempotencyPolicy
}

// HTTPClient executes HTTP requests for payment provider clients with production-ready defaults,
// automated exponential backoff retries, and centralized idempotency policy enforcement.
//
// Provider implementations should hold *HTTPClient as a pointer field rather than embedding
// it to avoid method promotion (preventing raw transport calls from leaking into domain APIs),
// protect internal mutex state from accidental copying, and share connection pools across services.
type HTTPClient struct {
	client            *http.Client
	logger            *slog.Logger
	dumpWriter        io.Writer
	dumpMu            sync.Mutex
	maxAttempts       int
	retryBaseDelay    time.Duration
	idempotencyPolicy IdempotencyPolicy
}

// NewHTTPClient creates a configured HTTP client suitable for payment gateway transport operations.
func NewHTTPClient(config HTTPClientConfig) *HTTPClient {
	connectionTimeout := config.ConnectionTimeout
	if connectionTimeout <= 0 {
		connectionTimeout = defaultConnectionTimeout
	}

	requestTimeout := config.RequestTimeout
	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}

	retryBaseDelay := config.RetryBaseDelay
	if retryBaseDelay <= 0 {
		retryBaseDelay = defaultRetryBaseDelay
	}

	maxAttempts := config.MaxAttempts
	if maxAttempts <= 0 || maxAttempts > defaultMaxAttempts {
		maxAttempts = defaultMaxAttempts
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout: connectionTimeout,
	}).DialContext

	if config.TLSConfig != nil {
		transport.TLSClientConfig = config.TLSConfig.Clone()
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout:   requestTimeout,
			Transport: transport,
		},
		logger:            config.Logger,
		dumpWriter:        config.DumpWriter,
		maxAttempts:       maxAttempts,
		retryBaseDelay:    retryBaseDelay,
		idempotencyPolicy: config.IdempotencyPolicy,
	}
}

// SetTransport replaces the underlying HTTP transport. This is primarily intended for
// testing environments where a test server's custom transport must be injected.
func (c *HTTPClient) SetTransport(rt http.RoundTripper) {
	c.client.Transport = rt
}

// Do sends req with ctx and returns the final HTTP response.
// It centralizes Idempotency-Key validation, auto-generation, exponential backoff retries,
// and audit logging.
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("paykit: nil http request")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.execute(ctx, req)
}

// DoRequest sends req using req.Context() as the execution context.
func (c *HTTPClient) DoRequest(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("paykit: nil http request")
	}
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return c.Do(ctx, req)
}

func (c *HTTPClient) execute(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c == nil {
		c = NewHTTPClient(HTTPClientConfig{})
	}
	if c.client == nil {
		c.client = NewHTTPClient(HTTPClientConfig{}).client
	}

	// Validate or inject Idempotency-Key on mutating methods
	if isMutatingMethod(req.Method) {
		key := req.Header.Get(headerIdempotencyKey)
		if key == "" {
			switch c.idempotencyPolicy {
			case IdempotencyPolicyRequire:
				return nil, ErrMissingIdempotencyKeySentinel
			case IdempotencyPolicyAutoGenerate:
				generatedKey, err := generateIdempotencyKey()
				if err != nil {
					return nil, fmt.Errorf("paykit: failed to generate idempotency key: %w", err)
				}
				req.Header.Set(headerIdempotencyKey, generatedKey)
			case IdempotencyPolicyOptional:
				// Proceed without key; request will not be eligible for retries
			}
		}
	}

	attempts := c.maxAttempts
	if attempts <= 0 || attempts > defaultMaxAttempts {
		attempts = defaultMaxAttempts
	}

	canRetry := isRetryableMethod(req)
	hasBody := req.Body != nil && req.Body != http.NoBody

	// When retries are possible and a body is present, GetBody must be non-nil
	if canRetry && attempts > 1 && hasBody && req.GetBody == nil {
		return nil, errors.New("paykit: request body cannot be replayed; set req.GetBody or use a single-attempt client")
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		var attemptReq *http.Request
		if attempt == 1 {
			attemptReq = req.Clone(ctx)
		} else {
			var err error
			attemptReq, err = cloneRequestForRetry(ctx, req)
			if err != nil {
				return nil, err
			}
		}

		c.dumpRequest(attemptReq)
		c.logDebug("sending http request",
			"method", attemptReq.Method,
			"url", attemptReq.URL.String(),
			"attempt", attempt,
		)

		resp, err := c.client.Do(attemptReq)
		if err == nil {
			c.dumpResponse(resp)
			c.logDebug("received http response",
				"method", attemptReq.Method,
				"url", attemptReq.URL.String(),
				"attempt", attempt,
				"status", resp.StatusCode,
			)

			// 429 Too Many Requests: honor Retry-After header
			if resp.StatusCode == http.StatusTooManyRequests {
				if !canRetry || attempt == attempts {
					return resp, nil
				}

				retryAfterHeader := resp.Header.Get("Retry-After")
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				delay, ok := parseRetryAfter(retryAfterHeader)
				if !ok {
					return nil, fmt.Errorf("paykit: 429 Too Many Requests with no Retry-After header")
				}

				if err := sleepFor(ctx, delay); err != nil {
					return nil, err
				}
				continue
			}

			if !shouldRetryResponse(resp) || !canRetry || attempt == attempts {
				return resp, nil
			}

			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		} else {
			lastErr = err
			c.logDebug("http request failed",
				"method", attemptReq.Method,
				"url", attemptReq.URL.String(),
				"attempt", attempt,
				"error", err,
			)

			if !shouldRetryError(err) || !canRetry || attempt == attempts {
				return nil, err
			}
		}

		if err := sleepBeforeRetry(ctx, c.retryBaseDelay, attempt); err != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("%w: previous request error: %v", err, lastErr)
			}
			return nil, err
		}
	}

	return nil, lastErr
}

func cloneRequestForRetry(ctx context.Context, req *http.Request) (*http.Request, error) {
	attemptReq := req.Clone(ctx)
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("paykit: clone request body: %w", err)
		}
		attemptReq.Body = body
	}
	return attemptReq, nil
}

// isMutatingMethod reports whether the HTTP method alters server state.
func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// isRetryableMethod reports whether req may be safely retried.
// Mutating methods (POST, PUT, PATCH, DELETE) are only retried when an Idempotency-Key
// header is present, guaranteeing upstream deduplication.
func isRetryableMethod(req *http.Request) bool {
	if isMutatingMethod(req.Method) {
		return req.Header.Get(headerIdempotencyKey) != ""
	}
	return true
}

func shouldRetryResponse(resp *http.Response) bool {
	return resp.StatusCode >= http.StatusInternalServerError
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		var certErr x509.CertificateInvalidError
		if errors.As(urlErr.Err, &certErr) {
			return false
		}
		var dnsErr *net.DNSError
		if errors.As(urlErr.Err, &dnsErr) && !dnsErr.Temporary() {
			return false
		}
	}
	return true
}

func parseRetryAfter(header string) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}
	if secs, err := strconv.ParseFloat(header, 64); err == nil && secs >= 0 {
		return time.Duration(secs * float64(time.Second)), true
	}
	if t, err := http.ParseTime(header); err == nil {
		delay := time.Until(t)
		if delay < 0 {
			delay = 0
		}
		return delay, true
	}
	return 0, false
}

func sleepBeforeRetry(ctx context.Context, baseDelay time.Duration, attempt int) error {
	return sleepFor(ctx, baseDelay*time.Duration(1<<uint(attempt-1)))
}

func sleepFor(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *HTTPClient) logDebug(message string, args ...any) {
	if c.logger != nil {
		c.logger.Debug(message, args...)
	}
}

func (c *HTTPClient) dumpRequest(req *http.Request) {
	if c.dumpWriter == nil {
		return
	}
	c.dumpMu.Lock()
	defer c.dumpMu.Unlock()

	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		c.logDebug("failed to dump http request", "error", err)
		return
	}
	_, _ = c.dumpWriter.Write(dump)
	_, _ = c.dumpWriter.Write([]byte("\n"))
}

func (c *HTTPClient) dumpResponse(resp *http.Response) {
	if c.dumpWriter == nil {
		return
	}
	c.dumpMu.Lock()
	defer c.dumpMu.Unlock()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		c.logDebug("failed to dump http response", "error", err)
		return
	}
	_, _ = c.dumpWriter.Write(dump)
	_, _ = c.dumpWriter.Write([]byte("\n"))
}

// generateIdempotencyKey generates an RFC 4122 compliant UUIDv4 using crypto/rand.
func generateIdempotencyKey() (string, error) {
	var u [16]byte
	if _, err := io.ReadFull(rand.Reader, u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4
	u[8] = (u[8] & 0x3f) | 0x80 // Variant RFC 4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16]), nil
}
