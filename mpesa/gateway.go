package mpesa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Flying-Tea-Squad/paykit-go"
)

const defaultMPESABaseURL = "https://sandbox.safaricom.co.ke"

// Config contains the configuration needed to construct an M-Pesa gateway.
type Config struct {
	BaseURL    string
	HTTPClient *paykit.HTTPClient

	ConsumerKey        string
	ConsumerSecret     string
	Passkey            string
	ShortCode          string
	InitiatorName      string
	SecurityCredential string
	ResultURL          string
	QueueTimeoutURL    string

	TokenManager AccessTokenProvider
}

// Gateway adapts the shared paykit interfaces to the M-Pesa Daraja API.
type Gateway struct {
	client *Client
	config Config
}

var _ paykit.Gateway = (*Gateway)(nil)
var _ paykit.Disburser = (*Gateway)(nil)

func init() {
	paykit.Register("mpesa", func() (paykit.Gateway, error) {
		cfg := Config{
			BaseURL:            os.Getenv("MPESA_BASE_URL"),
			ConsumerKey:        os.Getenv("MPESA_CONSUMER_KEY"),
			ConsumerSecret:     os.Getenv("MPESA_CONSUMER_SECRET"),
			Passkey:            os.Getenv("MPESA_PASSKEY"),
			ShortCode:          os.Getenv("MPESA_SHORTCODE"),
			InitiatorName:      os.Getenv("MPESA_INITIATOR_NAME"),
			SecurityCredential: os.Getenv("MPESA_SECURITY_CREDENTIAL"),
			ResultURL:          os.Getenv("MPESA_RESULT_URL"),
			QueueTimeoutURL:    os.Getenv("MPESA_QUEUE_TIMEOUT_URL"),
		}
		return New(cfg)
	})
}

// New creates a new configured M-Pesa gateway.
func New(cfg Config) (*Gateway, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultMPESABaseURL
	}

	passkey := strings.TrimSpace(cfg.Passkey)
	if passkey == "" {
		return nil, errors.New("mpesa: passkey is required")
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	}

	if cfg.TokenManager == nil {
		if strings.TrimSpace(cfg.ConsumerKey) == "" || strings.TrimSpace(cfg.ConsumerSecret) == "" {
			return nil, errors.New("mpesa: consumer key and consumer secret are required unless a custom token manager is provided")
		}
		cfg.TokenManager = NewTokenManager(cfg.HTTPClient, cfg.ConsumerKey, cfg.ConsumerSecret, baseURL)
	}

	return &Gateway{
		client: NewMpesaClient(baseURL, cfg.HTTPClient, passkey, cfg.TokenManager),
		config: cfg,
	}, nil
}

// NewMpesa is an alias for New and preserves the provider naming convention used throughout the ADRs.
func NewMpesa(cfg Config) (*Gateway, error) {
	return New(cfg)
}

// Name returns the canonical payment provider name.
func (g *Gateway) Name() string {
	return "mpesa"
}

// Charge initiates an STK Push payment request.
func (g *Gateway) Charge(ctx context.Context, req *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	if ctx == nil {
		return nil, errors.New("mpesa: nil context")
	}
	if req == nil {
		return nil, errors.New("mpesa: charge request cannot be nil")
	}
	if req.Amount.Amount() <= 0 {
		return nil, errors.New("mpesa: charge amount must be greater than zero")
	}
	if strings.TrimSpace(req.Phone) == "" {
		return nil, errors.New("mpesa: phone number is required")
	}
	if strings.TrimSpace(g.config.ShortCode) == "" {
		return nil, errors.New("mpesa: short code is required")
	}

	amount := int(req.Amount.Amount() / 100)
	if amount <= 0 {
		amount = 1
	}

	stkReq := STKPushRequest{
		IdempotencyKey:    req.IdempotencyKey,
		BusinessShortCode: g.config.ShortCode,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            normalizeMsisdn(req.Phone),
		PartyB:            g.config.ShortCode,
		PhoneNumber:       normalizeMsisdn(req.Phone),
		CallBackURL:       req.CallbackURL,
		AccountReference:  req.Reference,
		TransactionDesc:   req.Description,
	}

	resp, err := g.client.STKPush(ctx, stkReq)
	if err != nil {
		return nil, err
	}

	chargeResp := &paykit.ChargeResponse{
		Success:           resp.ResponseCode == "0",
		Message:           resp.CustomerMessage,
		TransactionID:     resp.CheckoutRequestID,
		CheckoutRequestID: resp.CheckoutRequestID,
		Status:            paykit.StatusPendingAction,
		Metadata: map[string]any{
			"provider":            "mpesa",
			"merchant_request_id": resp.MerchantRequestID,
			"checkout_request_id": resp.CheckoutRequestID,
		},
	}
	if chargeResp.TransactionID == "" {
		chargeResp.TransactionID = resp.MerchantRequestID
	}
	if chargeResp.Message == "" {
		chargeResp.Message = resp.ResponseDescription
	}
	if !chargeResp.Success {
		chargeResp.Status = paykit.StatusFailed
	}
	if raw, err := json.Marshal(resp); err == nil {
		chargeResp.Raw = raw
	}
	return chargeResp, nil
}

// QueryStatus checks the outcome of a previously initiated STK Push transaction.
func (g *Gateway) QueryStatus(ctx context.Context, req *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	if ctx == nil {
		return nil, errors.New("mpesa: nil context")
	}
	if req == nil {
		return nil, errors.New("mpesa: status request cannot be nil")
	}
	if strings.TrimSpace(req.TransactionID) == "" {
		return nil, errors.New("mpesa: checkout request ID is required")
	}
	if strings.TrimSpace(g.config.ShortCode) == "" {
		return nil, errors.New("mpesa: short code is required")
	}

	if g.client == nil || g.client.httpClient == nil {
		return nil, errors.New("mpesa: client is not initialized")
	}

	timestamp := generateTimestamp()
	password := generatePassword(g.config.ShortCode, g.client.passkey, timestamp)
	payload := map[string]any{
		"BusinessShortCode": g.config.ShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"CheckoutRequestID": req.TransactionID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("mpesa: marshal status request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.client.baseURL+"/mpesa/stkpushquery/v1/query", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("mpesa: create status query request: %w", err)
	}

	token, err := g.client.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	resp, err := g.client.httpClient.Do(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mpesa: status query failed with status %d", resp.StatusCode)
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("mpesa: decode status query response: %w", err)
	}

	code := providerCode(response["ResponseCode"])
	status := paykit.StatusUnknown
	if code == "0" {
		status = paykit.StatusSuccess
	} else if code != "" {
		status = paykit.StatusFailed
	}
	message := providerMessage(response["ResponseDescription"])
	if message == "" {
		message = strings.TrimSpace(req.TransactionID)
	}

	statusResp := &paykit.StatusResponse{
		Success:       code == "0",
		Message:       message,
		TransactionID: req.TransactionID,
		Status:        status,
		Metadata: map[string]any{
			"provider": "mpesa",
			"raw":      response,
		},
	}
	if raw, err := json.Marshal(response); err == nil {
		statusResp.Raw = raw
	}
	return statusResp, nil
}

// Refund initiates a reversal of a completed M-Pesa transaction.
func (g *Gateway) Refund(ctx context.Context, req *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	if ctx == nil {
		return nil, errors.New("mpesa: nil context")
	}
	if req == nil {
		return nil, errors.New("mpesa: refund request cannot be nil")
	}
	if strings.TrimSpace(req.TransactionID) == "" {
		return nil, errors.New("mpesa: transaction ID is required")
	}
	if strings.TrimSpace(g.config.ShortCode) == "" {
		return nil, errors.New("mpesa: short code is required")
	}
	if strings.TrimSpace(g.config.InitiatorName) == "" {
		return nil, errors.New("mpesa: initiator name is required")
	}
	if strings.TrimSpace(g.config.SecurityCredential) == "" {
		return nil, errors.New("mpesa: security credential is required")
	}

	amount := 0
	if req.Amount.Amount() > 0 {
		amount = int(req.Amount.Amount() / 100)
	}
	if amount <= 0 {
		amount = 1
	}

	payload := map[string]any{
		"Initiator":              g.config.InitiatorName,
		"SecurityCredential":     g.config.SecurityCredential,
		"CommandID":              "TransactionReversal",
		"TransactionID":          req.TransactionID,
		"Amount":                 amount,
		"ReceiverParty":          g.config.ShortCode,
		"ReceiverIdentifierType": "11",
		"Remarks":                req.Reason,
		"QueueTimeOutURL":        g.config.QueueTimeoutURL,
		"ResultURL":              g.config.ResultURL,
	}
	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("mpesa: marshal refund request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.client.baseURL+"/mpesa/reversal/v1/request", strings.NewReader(string(rawBody)))
	if err != nil {
		return nil, fmt.Errorf("mpesa: create refund request: %w", err)
	}

	token, err := g.client.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	resp, err := g.client.httpClient.Do(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mpesa: refund request failed with status %d", resp.StatusCode)
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("mpesa: decode refund response: %w", err)
	}

	code := providerCode(response["ResponseCode"])
	statusResp := &paykit.RefundResponse{
		Success:       code == "0",
		Message:       providerMessage(response["ResponseDescription"]),
		RefundID:      providerMessage(response["ConversationID"]),
		TransactionID: req.TransactionID,
		Status:        paykit.StatusProcessing,
		Metadata: map[string]any{
			"provider": "mpesa",
			"raw":      response,
		},
	}
	if statusResp.Message == "" {
		statusResp.Message = "refund initiated"
	}
	if code == "0" {
		statusResp.Status = paykit.StatusSuccess
	} else if code != "" {
		statusResp.Success = false
		statusResp.Status = paykit.StatusFailed
		if statusResp.Message == "" {
			statusResp.Message = providerMessage(response["ResponseDescription"])
		}
	}
	if raw, err := json.Marshal(response); err == nil {
		statusResp.Raw = raw
	}
	return statusResp, nil
}

// Disburse triggers a B2C disbursement to a recipient.
func (g *Gateway) Disburse(ctx context.Context, req *paykit.DisbursementRequest) (*paykit.DisbursementResponse, error) {
	if ctx == nil {
		return nil, errors.New("mpesa: nil context")
	}
	if req == nil {
		return nil, errors.New("mpesa: disbursement request cannot be nil")
	}
	if req.Amount.Amount() <= 0 {
		return nil, errors.New("mpesa: disbursement amount must be greater than zero")
	}
	if strings.TrimSpace(req.Phone) == "" {
		return nil, errors.New("mpesa: recipient phone number is required")
	}
	if strings.TrimSpace(g.config.ShortCode) == "" {
		return nil, errors.New("mpesa: short code is required")
	}
	if strings.TrimSpace(g.config.InitiatorName) == "" {
		return nil, errors.New("mpesa: initiator name is required")
	}
	if strings.TrimSpace(g.config.SecurityCredential) == "" {
		return nil, errors.New("mpesa: security credential is required")
	}

	amount := int(req.Amount.Amount() / 100)
	if amount <= 0 {
		amount = 1
	}

	payload := B2CRequest{
		IdempotencyKey:     req.IdempotencyKey,
		InitiatorName:      g.config.InitiatorName,
		SecurityCredential: g.config.SecurityCredential,
		CommandID:          "BusinessPayment",
		Amount:             amount,
		PartyA:             g.config.ShortCode,
		PartyB:             normalizeMsisdn(req.Phone),
		Remarks:            req.Description,
		QueueTimeOutURL:    g.config.QueueTimeoutURL,
		ResultURL:          g.config.ResultURL,
		Occasion:           req.Reference,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("mpesa: marshal disbursement request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.client.baseURL+"/mpesa/b2c/v1/paymentrequest", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("mpesa: create disbursement request: %w", err)
	}

	token, err := g.client.tokenManager.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	resp, err := g.client.httpClient.Do(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("mpesa: disbursement request failed with status %d", resp.StatusCode)
	}

	var result B2CResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("mpesa: decode disbursement response: %w", err)
	}

	outcome := &paykit.DisbursementResponse{
		Success:       result.ResponseCode == "0",
		Message:       result.ResponseDescription,
		TransactionID: result.ConversationID,
		Status:        paykit.StatusProcessing,
		Metadata: map[string]any{
			"provider":                   "mpesa",
			"originator_conversation_id": result.OriginatorConversationID,
		},
	}
	if outcome.TransactionID == "" {
		outcome.TransactionID = result.OriginatorConversationID
	}
	if outcome.Success {
		outcome.Status = paykit.StatusSuccess
	} else if result.ResponseCode != "" {
		outcome.Status = paykit.StatusFailed
	}
	if raw, err := json.Marshal(result); err == nil {
		outcome.Raw = raw
	}
	return outcome, nil
}

func normalizeMsisdn(phone string) string {
	trimmed := strings.TrimSpace(phone)
	trimmed = strings.TrimPrefix(trimmed, "+")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	trimmed = strings.ReplaceAll(trimmed, "-", "")
	if strings.HasPrefix(trimmed, "0") && len(trimmed) == 10 {
		return "254" + trimmed[1:]
	}
	return trimmed
}

func providerCode(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case json.Number:
		return value.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", value)
	}
}

func providerMessage(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", value)
	}
}
