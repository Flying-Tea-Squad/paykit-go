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

// STKPushQuery checks the processing state of an STK Push transaction with Safaricom Daraja.
//
// Callers supply the CheckoutRequestID returned by STKPush. If Timestamp or Password
// are omitted in req, they are automatically computed using the client's passkey.
func (c *Client) STKPushQuery(ctx context.Context, req STKPushQueryRequest) (*STKPushQueryResponse, error) {
	if strings.TrimSpace(req.CheckoutRequestID) == "" {
		return nil, errors.New("mpesa: checkout request ID is required for STK push query")
	}
	if strings.TrimSpace(req.BusinessShortCode) == "" {
		return nil, errors.New("mpesa: business short code is required for STK push query")
	}

	if req.Timestamp == "" {
		req.Timestamp = generateTimestamp()
	}
	if req.Password == "" {
		if c.passkey == "" {
			return nil, errors.New("mpesa: passkey is required to generate STK query password")
		}
		req.Password = generatePassword(
			req.BusinessShortCode,
			c.passkey,
			req.Timestamp,
		)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to encode STK query request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/mpesa/stkpushquery/v1/query"
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to create STK query request: %w", err)
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
		return nil, fmt.Errorf("mpesa: STK query request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mpesa: unexpected status code %d from STK query", resp.StatusCode)
	}

	var queryResp STKPushQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("mpesa: failed to decode STK query response: %w", err)
	}

	return &queryResp, nil
}
