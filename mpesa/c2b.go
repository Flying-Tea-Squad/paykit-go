package mpesa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Flying-Tea-Squad/paykit-go"
)

// ResponseType constants for C2B URL registration.
const (
	ResponseTypeCompleted = "Completed"
	ResponseTypeCancelled = "Cancelled"
)

// CommandID constants for C2B sandbox simulation.
const (
	CommandIDCustomerPayBillOnline  = "CustomerPayBillOnline"
	CommandIDCustomerBuyGoodsOnline = "CustomerBuyGoodsOnline"
)

// C2BRegisterURL registers the validation and confirmation URLs for a C2B short code.
//
// Validation rules:
//   - ShortCode, ConfirmationURL, and ValidationURL must be non-empty.
//   - ResponseType defaults to "Completed" if empty (valid values: "Completed" or "Cancelled").
//
// The request sends an authenticated POST to /mpesa/c2b/v1/registerurl.
func (c *Client) C2BRegisterURL(ctx context.Context, req C2BRegisterRequest) (*C2BRegisterResponse, error) {
	if strings.TrimSpace(req.ShortCode) == "" {
		return nil, errors.New("mpesa: short code is required")
	}
	if strings.TrimSpace(req.ConfirmationURL) == "" {
		return nil, errors.New("mpesa: confirmation URL is required")
	}
	if strings.TrimSpace(req.ValidationURL) == "" {
		return nil, errors.New("mpesa: validation URL is required")
	}

	if req.ResponseType == "" {
		req.ResponseType = ResponseTypeCompleted
	} else if req.ResponseType != ResponseTypeCompleted && req.ResponseType != ResponseTypeCancelled {
		return nil, fmt.Errorf("mpesa: invalid response type %q, must be 'Completed' or 'Cancelled'", req.ResponseType)
	}

	var resp C2BRegisterResponse
	if err := c.doAuthenticatedPost(ctx, "/mpesa/c2b/v1/registerurl", req, req.IdempotencyKey, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// C2BSimulate sends a transaction simulation request to the Daraja C2B sandbox.
//
// Validation rules:
//   - ShortCode and Msisdn must be non-empty.
//   - Amount must be greater than zero.
//   - CommandID defaults to "CustomerPayBillOnline" if empty (valid values: "CustomerPayBillOnline" or "CustomerBuyGoodsOnline").
//
// The request sends an authenticated POST to /mpesa/c2b/v1/simulate.
func (c *Client) C2BSimulate(ctx context.Context, req C2BSimulateRequest) (*C2BSimulateResponse, error) {
	if strings.TrimSpace(req.ShortCode) == "" {
		return nil, errors.New("mpesa: short code is required")
	}
	if strings.TrimSpace(req.Msisdn) == "" {
		return nil, errors.New("mpesa: MSISDN is required")
	}
	if req.Amount <= 0 {
		return nil, errors.New("mpesa: amount must be greater than zero")
	}

	if req.CommandID == "" {
		req.CommandID = CommandIDCustomerPayBillOnline
	} else if req.CommandID != CommandIDCustomerPayBillOnline && req.CommandID != CommandIDCustomerBuyGoodsOnline {
		return nil, fmt.Errorf("mpesa: invalid command ID %q, must be 'CustomerPayBillOnline' or 'CustomerBuyGoodsOnline'", req.CommandID)
	}

	var resp C2BSimulateResponse
	if err := c.doAuthenticatedPost(ctx, "/mpesa/c2b/v1/simulate", req, req.IdempotencyKey, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) doAuthenticatedPost(
	ctx context.Context,
	path string,
	reqBody any,
	idempotencyKey string,
	result any,
) error {
	if c == nil {
		return errors.New("mpesa: client is nil")
	}
	if c.tokenManager == nil {
		return errors.New("mpesa: token manager is required")
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("mpesa: failed to encode request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + path
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("mpesa: failed to create request: %w", err)
	}

	token, err := c.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("mpesa: failed to get access token: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	if idempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", idempotencyKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	}

	resp, err := httpClient.Do(ctx, httpReq)
	if err != nil {
		return fmt.Errorf("mpesa: request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if msg := strings.TrimSpace(string(body)); msg != "" {
			return fmt.Errorf("mpesa: unexpected status code %d: %s", resp.StatusCode, msg)
		}
		return fmt.Errorf("mpesa: unexpected status code %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("mpesa: failed to decode response: %w", err)
		}
	}

	return nil
}
