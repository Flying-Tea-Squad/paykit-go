package mpesa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

// B2CPayment sends money from a business account to a customer's M-Pesa account.
func (c *Client) B2CPayment(
	ctx context.Context,
	req *B2CRequest,
) (*paykit.Response, error) {
	// Ensure a token manager is configured before making authenticated requests.
	if c.tokenManager == nil {
		return nil, fmt.Errorf("mpesa: token manager is required")
	}

	// Retrieve a valid OAuth access token.
	token, err := c.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to get access token: %w", err)
	}

	// Convert the B2C request into JSON format.
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to encode B2C request: %w", err)
	}

	// Create request body from JSON.
	requestBody := bytes.NewBuffer(body)

	// Build the M-Pesa B2C endpoint URL.
	endpoint := strings.TrimRight(c.baseURL, "/") +
		"/mpesa/b2c/v1/paymentrequest"

	// Create HTTP POST request.
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to create B2C request: %w", err)
	}

	// Add OAuth authentication header.
	httpReq.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	// Tell M-Pesa the request body is JSON.
	httpReq.Header.Set(
		"Content-Type",
		"application/json",
	)

	// Execute HTTP request.
	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: B2C request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body.
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to read B2C response: %w", err)
	}

	// Decode M-Pesa response.
	var b2cResponse B2CResponse

	err = json.Unmarshal(responseBody, &b2cResponse)
	if err != nil {
		return nil, fmt.Errorf("mpesa: failed to decode B2C response: %w", err)
	}

	// Successful M-Pesa transactions return ResponseCode "0".
	success := resp.StatusCode == http.StatusOK &&
		b2cResponse.ResponseCode == "0"

	// Convert M-Pesa response into PayKit response format.
	return &paykit.Response{
		Success:       success,
		Message:       b2cResponse.ResponseDescription,
		TransactionID: b2cResponse.ConversationID,
		ErrorCode:     b2cResponse.ResponseCode,
		Raw:           responseBody,
		Metadata: map[string]any{
			"originatorConversationID": b2cResponse.OriginatorConversationID,
		},
	}, nil
}
