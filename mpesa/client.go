package mpesa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Flying-Tea-Squad/paykit-go"
)

// maxSTKPushResponseBytes bounds how much of an STK Push response body the client
// will read into memory (1 MiB), guarding against oversized upstream payloads.
const maxSTKPushResponseBytes = 1 << 20

// normalizeBaseURL canonicalizes a provider base URL: it trims surrounding
// whitespace, defaults a missing scheme to https, requires an http(s) scheme
// with a host, and strips trailing slashes so endpoint paths can be appended
// with a single "/".
func normalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("mpesa: base URL is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("mpesa: invalid base URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("mpesa: base URL scheme %q is not supported, use http or https", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("mpesa: base URL %q is missing a host", raw)
	}
	return strings.TrimRight(trimmed, "/"), nil
}

// validateSTKPushRequest ensures all fields required by the Daraja STK Push
// endpoint are present and well-formed before any network I/O occurs.
func validateSTKPushRequest(req STKPushRequest) error {
	if strings.TrimSpace(req.BusinessShortCode) == "" {
		return errors.New("mpesa: business short code is required for STK push")
	}
	if req.Amount <= 0 {
		return errors.New("mpesa: amount must be greater than zero for STK push")
	}
	if strings.TrimSpace(req.PartyA) == "" {
		return errors.New("mpesa: party A is required for STK push")
	}
	if strings.TrimSpace(req.PartyB) == "" {
		return errors.New("mpesa: party B is required for STK push")
	}
	if strings.TrimSpace(req.PhoneNumber) == "" {
		return errors.New("mpesa: phone number is required for STK push")
	}
	if !paykit.IsValidKenyanPhone(req.PhoneNumber) {
		return fmt.Errorf("mpesa: phone number %q is not a valid Kenyan phone number", req.PhoneNumber)
	}
	if strings.TrimSpace(req.CallBackURL) == "" {
		return errors.New("mpesa: callback URL is required for STK push")
	}
	return nil
}

// AccessTokenProvider supplies OAuth access tokens for M-Pesa API requests.
type AccessTokenProvider interface {
	GetAccessToken(ctx context.Context) (string, error)
}

// Client sends requests to the M-Pesa API.
type Client struct {
	baseURL string
	// httpClient is held as an unexported pointer rather than embedded to encapsulate transport
	// methods, avoid copying internal mutexes, and share connection pools with token managers.
	httpClient   *paykit.HTTPClient
	passkey      string
	tokenManager AccessTokenProvider
}

// NewMpesaClient creates an M-Pesa API client.
func NewMpesaClient(
	baseURL string,
	httpClient *paykit.HTTPClient,
	passkey string,
	tokenManager AccessTokenProvider,
) *Client {
	if httpClient == nil {
		// Match mpesa.New in gateway.go: mutating requests without an
		// Idempotency-Key are allowed rather than rejected.
		httpClient = paykit.NewHTTPClient(paykit.HTTPClientConfig{
			IdempotencyPolicy: paykit.IdempotencyPolicyOptional,
		})
	}

	// Canonicalize the base URL when possible; invalid values are preserved so
	// request-time validation can surface a descriptive error.
	if normalized, err := normalizeBaseURL(baseURL); err == nil {
		baseURL = normalized
	}

	return &Client{
		baseURL:      baseURL,
		httpClient:   httpClient,
		passkey:      passkey,
		tokenManager: tokenManager,
	}
}

// STKPush sends an STK Push payment request to the M-Pesa API.
//
// Callers may pre-supply Timestamp and Password; when omitted they are computed
// from the client's passkey, matching STKPushQuery.
func (c *Client) STKPush(ctx context.Context, req STKPushRequest) (*STKPushResponse, error) {
	if err := validateSTKPushRequest(req); err != nil {
		return nil, err
	}

	if req.Timestamp == "" {
		req.Timestamp = generateTimestamp()
	}
	if req.Password == "" {
		if c.passkey == "" {
			return nil, errors.New("mpesa: passkey is required to generate STK push password")
		}
		req.Password = generatePassword(
			req.BusinessShortCode,
			c.passkey,
			req.Timestamp,
		)
	}

	baseURL, err := normalizeBaseURL(c.baseURL)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to encode STK push request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/mpesa/stkpush/v1/processrequest",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to create STK push request: %w", err)
	}

	if c.tokenManager == nil {
		return nil, errors.New("mpesa: token manager is required")
	}

	token, err := c.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to get access token: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)

	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: STK push request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if msg := strings.TrimSpace(string(body)); msg != "" {
			return nil, fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Bound the response read so an oversized upstream payload cannot exhaust memory.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxSTKPushResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to read STK push response: %w", err)
	}
	if len(respBody) > maxSTKPushResponseBytes {
		return nil, fmt.Errorf("mpesa: STK push response exceeds the %d byte limit", maxSTKPushResponseBytes)
	}

	var stkResp STKPushResponse
	if err := json.Unmarshal(respBody, &stkResp); err != nil {
		return nil, fmt.Errorf("mpesa: failed to decode STK push response: %w", err)
	}

	return &stkResp, nil
}
