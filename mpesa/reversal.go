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

// ReverseTransaction sends a transaction reversal request to Safaricom M-Pesa.
//
// This operation is asynchronous; Daraja acknowledges receipt immediately with a
// ConversationID and OriginatorConversationID, with final reversal status reported
// to ResultURL or QueueTimeOutURL.
func (c *Client) ReverseTransaction(ctx context.Context, req ReversalRequest) (*ReversalResponse, error) {
	if strings.TrimSpace(req.Initiator) == "" {
		return nil, errors.New("mpesa: initiator is required for transaction reversal")
	}
	if strings.TrimSpace(req.SecurityCredential) == "" {
		return nil, errors.New("mpesa: security credential is required for transaction reversal")
	}
	if strings.TrimSpace(req.TransactionID) == "" {
		return nil, errors.New("mpesa: transaction ID is required for transaction reversal")
	}
	if strings.TrimSpace(req.ReceiverParty) == "" {
		return nil, errors.New("mpesa: receiver party is required for transaction reversal")
	}
	if req.Amount <= 0 {
		return nil, errors.New("mpesa: amount must be greater than zero for transaction reversal")
	}

	if req.CommandID == "" {
		req.CommandID = "TransactionReversal"
	}
	if req.RecieverIdentifierType == "" {
		req.RecieverIdentifierType = "11" // ShortCode
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to encode reversal request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/mpesa/reversal/v1/request"
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to create reversal request: %w", err)
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
		return nil, fmt.Errorf("mpesa: reversal request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mpesa: unexpected status code %d from reversal", resp.StatusCode)
	}

	var revResp ReversalResponse
	if err := json.NewDecoder(resp.Body).Decode(&revResp); err != nil {
		return nil, fmt.Errorf("mpesa: failed to decode reversal response: %w", err)
	}

	return &revResp, nil
}
