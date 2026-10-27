package mpesa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// B2CPayment sends money from an organization business account to a customer's M-Pesa account.
//
// The request is transmitted securely using the shared HTTP transport, ensuring OAuth Bearer
// authentication, automatic retries for transient errors, and centralized idempotency handling.
func (c *Client) B2CPayment(ctx context.Context, req B2CRequest) (*B2CResponse, error) {
	if strings.TrimSpace(req.InitiatorName) == "" {
		return nil, errors.New("mpesa: initiator name is required for B2C payment")
	}
	if strings.TrimSpace(req.SecurityCredential) == "" {
		return nil, errors.New("mpesa: security credential is required for B2C payment")
	}
	if strings.TrimSpace(req.PartyA) == "" {
		return nil, errors.New("mpesa: party A (sender short code) is required for B2C payment")
	}
	if strings.TrimSpace(req.PartyB) == "" {
		return nil, errors.New("mpesa: party B (recipient phone) is required for B2C payment")
	}
	if req.Amount <= 0 {
		return nil, errors.New("mpesa: amount must be greater than zero for B2C payment")
	}

	if req.CommandID == "" {
		req.CommandID = "BusinessPayment"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to encode B2C request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/mpesa/b2c/v1/paymentrequest"
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to create B2C request: %w", err)
	}

	if c.tokenManager == nil {
		return nil, errors.New("mpesa: token manager is required")
	}

	token, err := c.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to get access token: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	resp, err := c.httpClient.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: B2C request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mpesa: unexpected status code %d from B2C payment", resp.StatusCode)
	}

	var b2cResp B2CResponse
	if err := json.NewDecoder(resp.Body).Decode(&b2cResp); err != nil {
		return nil, fmt.Errorf("mpesa: failed to decode B2C response: %w", err)
	}

	return &b2cResp, nil
}
