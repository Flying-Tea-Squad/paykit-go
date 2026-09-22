package paykit

import (
	"context"
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
	"time"
)

const (
	defaultConnectionTimeout = 10 * time.Second
	defaultRequestTimeout    = 30 * time.Second
	defaultRetryBaseDelay    = 100 * time.Millisecond
	defaultMaxAttempts       = 3
)

// HTTPClient wraps net/http with SDK defaults for timeouts, retries, logging,
// and optional wire dumps.
type HTTPClient struct {
	client         *http.Client
	logger         *slog.Logger
	dumpWriter     io.Writer
	maxAttempts    int
	retryBaseDelay time.Duration
}

// HTTPClientConfig configures HTTPClient.
type HTTPClientConfig struct {
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
	TLSConfig         *tls.Config
	Logger            *slog.Logger
	DumpWriter        io.Writer
	RetryBaseDelay    time.Duration
	MaxAttempts       int
}

// NewHTTPClient creates an HTTPClient with sensible defaults.
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
		logger:         config.Logger,
		dumpWriter:     config.DumpWriter,
		maxAttempts:    maxAttempts,
		retryBaseDelay: retryBaseDelay,
	}
}

// SetTransport replaces the underlying HTTP transport. This is intended for
// testing, where a test server's transport (carrying its TLS certificates)
// must be injected after construction.
func (c *HTTPClient) SetTransport(rt http.RoundTripper) {
	c.client.Transport = rt
}

// Do sends req with ctx and returns the final HTTP response.
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c == nil {
		c = NewHTTPClient(HTTPClientConfig{})
	}
	if c.client == nil {
		c.client = NewHTTPClient(HTTPClientConfig{}).client
	}
	if req == nil {
		return nil, fmt.Errorf("paykit: nil http request")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	attempts := c.maxAttempts
	if attempts <= 0 || attempts > defaultMaxAttempts {
		attempts = defaultMaxAttempts
	}

	// Non-idempotent methods (POST, PATCH) must not be retried automatically
	// unless the caller has signalled safety via an Idempotency-Key header.
	canRetry := isRetryableMethod(req)

	// Body replay without buffering the full payload requires GetBody.
	// net/http.NewRequest already sets GetBody for strings.Reader, bytes.Reader,
	// and bytes.Buffer. For any other body type callers must set it themselves.
	// When the method is non-idempotent (canRetry == false) the body is sent
	// exactly once, so GetBody is not needed.
	if canRetry && attempts > 1 && req.Body != nil && req.GetBody == nil {
		return nil, errors.New("paykit: request body cannot be replayed; set req.GetBody or use a single-attempt client")
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptReq, err := cloneRequestForAttempt(ctx, req)
		if err != nil {
			return nil, err
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

			// 429 Too Many Requests: honor the Retry-After header when present,
			// otherwise surface an error instead of retrying blindly.
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

func cloneRequestForAttempt(ctx context.Context, req *http.Request) (*http.Request, error) {
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

// isRetryableMethod reports whether req may be safely retried.
// POST and PATCH are non-idempotent; they are only retried when the caller
// provides an Idempotency-Key header, signalling that the server can deduplicate.
func isRetryableMethod(req *http.Request) bool {
	switch req.Method {
	case http.MethodPost, http.MethodPatch:
		return req.Header.Get("Idempotency-Key") != ""
	default:
		return true
	}
}

// shouldRetryResponse reports whether a successful HTTP response warrants a retry.
// 429 Too Many Requests is handled separately via parseRetryAfter.
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

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		c.logDebug("failed to dump http response", "error", err)
		return
	}

	_, _ = c.dumpWriter.Write(dump)
	_, _ = c.dumpWriter.Write([]byte("\n"))

}
