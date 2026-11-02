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
	"math/big"
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
	defaultConnectionTimeout  = 10 * time.Second
	defaultRequestTimeout     = 30 * time.Second
	defaultRetryBaseDelay     = 100 * time.Millisecond
	defaultMaxRetryAfterDelay = 60 * time.Second
	defaultMaxAttempts        = 3
	headerIdempotencyKey      = "Idempotency-Key"
)

// HTTPClientConfig configures the shared transport client.
type HTTPClientConfig struct {
	// ConnectionTimeout specifies the maximum duration waiting for a network dial to complete.
	// Defaults to 10 seconds.
	ConnectionTimeout time.Duration

	// RequestTimeout specifies a time limit for requests made by this client.
	// Defaults to 30 seconds.
	RequestTimeout time.Duration

	// TLSConfig contains custom TLS client configuration. It is cloned to prevent concurrent mutations.
	TLSConfig *tls.Config

	// Logger is an optional structured logger. When provided, debug details of request/response attempts are recorded.
	Logger *slog.Logger

	// DumpWriter is an optional writer to which full wire dumps of requests and responses will be written.
	DumpWriter io.Writer

	// RetryBaseDelay is the initial backoff delay between retry attempts.
	// Defaults to 100 milliseconds.
	RetryBaseDelay time.Duration

	// MaxAttempts is the maximum number of attempts (initial attempt + retries) for retryable operations.
	// Values greater than 3 are clamped to 3. Defaults to 3.
	MaxAttempts int

	// IdempotencyPolicy specifies the handling strategy for mutating requests missing an Idempotency-Key header.
	// Defaults to IdempotencyPolicyRequire.
	IdempotencyPolicy IdempotencyPolicy
}

// HTTPClient executes HTTP requests for payment provider clients with production-ready defaults,
// automated exponential backoff retries, and centralized idempotency policy enforcement.
type HTTPClient struct {
	// Provider implementations hold *HTTPClient as a pointer field rather than embedding
	// to avoid method promotion, prevent mutex copying, and share connection pools across services.
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
// It centralizes Idempotency-Key validation, auto-generation, exponential backoff retries with jitter,
// and audit logging.
//
// Side Effects & Caller Responsibilities:
//   - req.Body is consumed and closed upon request transmission.
//   - If req has a body and is eligible for retry (idempotent methods or requests
//     carrying an Idempotency-Key header), req.GetBody must be non-nil so the
//     payload can be replayed; otherwise Do returns an error before dispatch.
//     Callers constructing requests with standard constructors like http.NewRequestWithContext
//     using bytes.Reader, bytes.Buffer, or strings.Reader will have req.GetBody automatically populated.
//   - The caller is responsible for closing the returned resp.Body when err is nil.
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
	client := c.client
	if client == nil {
		client = NewHTTPClient(HTTPClientConfig{}).client
	}

	var idempotencyKey string
	if isMutatingMethod(req.Method) {
		idempotencyKey = req.Header.Get(headerIdempotencyKey)
		if idempotencyKey == "" {
			switch c.idempotencyPolicy {
			case IdempotencyPolicyRequire:
				return nil, ErrMissingIdempotencyKeySentinel
			case IdempotencyPolicyAutoGenerate:
				key, err := generateIdempotencyKey()
				if err != nil {
					return nil, fmt.Errorf("paykit: failed to generate idempotency key: %w", err)
				}
				idempotencyKey = key
			case IdempotencyPolicyOptional:
				// Proceed without key; request will not be eligible for retries
			}
		}
	}

	attempts := c.maxAttempts
	if attempts <= 0 || attempts > defaultMaxAttempts {
		attempts = defaultMaxAttempts
	}

	retryBaseDelay := c.retryBaseDelay
	if retryBaseDelay <= 0 {
		retryBaseDelay = defaultRetryBaseDelay
	}

	canRetry := isRetryableMethodWithKey(req.Method, idempotencyKey)
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

		if idempotencyKey != "" {
			attemptReq.Header.Set(headerIdempotencyKey, idempotencyKey)
		}

		c.dumpRequest(attemptReq)
		c.logDebug("sending http request",
			"method", attemptReq.Method,
			"url", attemptReq.URL.String(),
			"attempt", attempt,
		)

		resp, err := client.Do(attemptReq)
		if err == nil {
			c.dumpResponse(resp)
			c.logDebug("received http response",
				"method", attemptReq.Method,
				"url", attemptReq.URL.String(),
				"attempt", attempt,
				"status", resp.StatusCode,
			)

			// 429 Too Many Requests: honor Retry-After header or fall back to jittered backoff
			if resp.StatusCode == http.StatusTooManyRequests {
				if !canRetry || attempt == attempts {
					return resp, nil
				}

				retryAfterHeader := resp.Header.Get("Retry-After")
				delay, ok := parseRetryAfter(retryAfterHeader)
				if !ok {
					delay = calculateBackoffWithJitter(retryBaseDelay, attempt)
				} else if delay > defaultMaxRetryAfterDelay {
					delay = defaultMaxRetryAfterDelay
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

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

		if err := sleepBeforeRetry(ctx, retryBaseDelay, attempt); err != nil {
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
	return isRetryableMethodWithKey(req.Method, req.Header.Get(headerIdempotencyKey))
}

func isRetryableMethodWithKey(method, idempotencyKey string) bool {
	if isMutatingMethod(method) {
		return idempotencyKey != ""
	}
	return true
}

func shouldRetryResponse(resp *http.Response) bool {
	return resp.StatusCode >= http.StatusInternalServerError
}

// shouldRetryError reports whether a transport error warrants a retry.
// Context cancellation, deadline expiry, TLS certificate errors, and permanent
// DNS failures are not retried because they will not resolve on the next attempt.
func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// TLS certificate errors are configuration problems, not transient failures.
		var certErr x509.CertificateInvalidError
		if errors.As(urlErr.Err, &certErr) {
			return false
		}
		// Permanent DNS failures (e.g. unknown host) will not change on retry.
		var dnsErr *net.DNSError
		if errors.As(urlErr.Err, &dnsErr) && !dnsErr.Temporary() {
			return false
		}
	}
	return true
}

// parseRetryAfter parses the value of a Retry-After response header.
// It supports both delay-seconds (e.g. "120") and HTTP-date formats.
// Returns false if the header is empty or cannot be parsed.
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
	return sleepFor(ctx, calculateBackoffWithJitter(baseDelay, attempt))
}

func calculateBackoffWithJitter(baseDelay time.Duration, attempt int) time.Duration {
	if baseDelay <= 0 {
		return 0
	}
	if attempt < 1 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 30 {
		shift = 30
	}
	base := baseDelay * time.Duration(1<<uint(shift))

	// ±20% jitter: range is [0.8 * base, 1.2 * base]
	minDelay := base * 80 / 100
	maxDelay := base * 120 / 100
	variance := maxDelay - minDelay
	if variance <= 0 {
		return base
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(variance)+1))
	if err != nil {
		return base
	}
	return minDelay + time.Duration(n.Int64())
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
