package paykit

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
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

	if err := makeRequestBodyReplayable(req); err != nil {
		return nil, err
	}

	attempts := c.maxAttempts
	if attempts <= 0 || attempts > defaultMaxAttempts {
		attempts = defaultMaxAttempts
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

			if !shouldRetryResponse(resp) || attempt == attempts {
				return resp, nil
			}

			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		} else {
			lastErr = err
			c.logDebug("http request failed",
				"method", attemptReq.Method,
				"url", attemptReq.URL.String(),
				"attempt", attempt,
				"error", err,
			)

			if !shouldRetryError(err) || attempt == attempts {
				return nil, err
			}
		}

		if err := sleepBeforeRetry(ctx, c.retryBaseDelay, attempt); err != nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, err
		}
	}

	return nil, lastErr
}

func makeRequestBodyReplayable(req *http.Request) error {
	if req.Body == nil || req.GetBody != nil {
		return nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("paykit: read request body: %w", err)
	}
	if err := req.Body.Close(); err != nil {
		return fmt.Errorf("paykit: close request body: %w", err)
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))

	return nil
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

func shouldRetryResponse(resp *http.Response) bool {
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
}

func shouldRetryError(err error) bool {
	return err != nil
}

func sleepBeforeRetry(ctx context.Context, baseDelay time.Duration, attempt int) error {
	delay := baseDelay * time.Duration(1<<uint(attempt-1))
	timer := time.NewTimer(delay)
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

	c.dumpWriter.Write(dump)
	c.dumpWriter.Write([]byte("\n"))
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

	c.dumpWriter.Write(dump)
	c.dumpWriter.Write([]byte("\n"))
}
