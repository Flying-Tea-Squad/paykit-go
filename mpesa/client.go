package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Flying-Tea-Squad/paykit-go"
)

// AccessTokenProvider supplies OAuth access tokens for M-Pesa API requests.
type AccessTokenProvider interface {
	GetAccessToken(ctx context.Context) (string, error)
}

// Client sends requests to the M-Pesa API.
type Client struct {
	baseURL string
	paykit.HTTPClient
	passkey      string
	tokenManager AccessTokenProvider
}

// NewMpesaClient creates an M-Pesa API client.
func NewMpesaClient(
	baseURL string,
	httpClient paykit.HTTPClient,
	passkey string,
	tokenManager AccessTokenProvider,
) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:      baseURL,
		HTTPClient:   httpClient,
		passkey:      passkey,
		tokenManager: tokenManager,
	}
}

// STKPush sends an STK Push payment request to the M-Pesa API.
func (c *Client) STKPush(ctx context.Context, req STKPushRequest) (*STKPushResponse, error) {
	req.Timestamp = generateTimestamp()
	req.Password = generatePassword(
		req.BusinessShortCode,
		c.passkey,
		req.Timestamp,
	)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/mpesa/stkpush/v1/processrequest",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	if c.tokenManager == nil {
		return nil, fmt.Errorf("mpesa: token manager is required")
	}

	token, err := c.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)

	httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var stkResp STKPushResponse

	err = json.NewDecoder(resp.Body).Decode(&stkResp)
	if err != nil {
		return nil, err
	}

	return &stkResp, nil
}

func generateTimestamp() string {
	return time.Now().Format("20060102150405")
}

func generatePassword(shortcode, passkey, timeStamp string) string {
	raw := shortcode + passkey + timeStamp

	return base64.StdEncoding.EncodeToString([]byte(raw))
}
