package mpesa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Flying-Tea-Squad/paykit-go"
)

// Environment specifies the target deployment environment for Safaricom Daraja endpoints.
type Environment string

const (
	// EnvironmentSandbox targets the Safaricom Daraja sandbox environment.
	EnvironmentSandbox Environment = "sandbox"

	// EnvironmentProduction targets the Safaricom Daraja live production environment.
	EnvironmentProduction Environment = "production"
)

// Config encapsulates credentials and default settings for initializing an MpesaGateway.
type Config struct {
	// ConsumerKey is the Daraja API application key.
	ConsumerKey string

	// ConsumerSecret is the Daraja API application secret.
	ConsumerSecret string

	// Passkey is the Lipa Na M-Pesa Online passkey for STK Push requests.
	Passkey string

	// BusinessShortCode is the primary organization shortcode (Paybill or Till number).
	BusinessShortCode string

	// ShortCode is an alias for BusinessShortCode.
	ShortCode string

	// Environment specifies whether to use sandbox or live endpoints (defaults to EnvironmentSandbox).
	Environment Environment

	// BaseURL optionally overrides the endpoint URL (useful for mocking and integration tests).
	BaseURL string

	// CallbackURL is the default webhook URL where Daraja delivers STK Push transaction callbacks.
	CallbackURL string

	// InitiatorName is the organization operator username authorized for B2C and Reversal operations.
	InitiatorName string

	// SecurityCredential is the encrypted operator password for B2C and Reversal operations.
	SecurityCredential string

	// QueueTimeOutURL is the callback URL for timeouts during B2C or Reversal requests.
	QueueTimeOutURL string

	// ResultURL is the callback URL for asynchronous results of B2C or Reversal requests.
	ResultURL string

	// HTTPClient is the underlying HTTP transport client. If nil, a default paykit.HTTPClient is created.
	HTTPClient *paykit.HTTPClient
}

// Gateway implements paykit.Gateway and paykit.Disburser for Safaricom M-Pesa Daraja 2.0.
type Gateway struct {
	client *Client
	cfg    Config
}

// MpesaGateway is an alias for Gateway, providing alternative name discovery.
type MpesaGateway = Gateway

var (
	_ paykit.Gateway   = (*Gateway)(nil)
	_ paykit.Disburser = (*Gateway)(nil)
)

// New initializes and returns a new M-Pesa Gateway instance configured with the provided options.
//
// Required fields in cfg: ConsumerKey, ConsumerSecret, Passkey, and BusinessShortCode (or ShortCode).
func New(cfg Config) (*Gateway, error) {
	if cfg.BusinessShortCode == "" && cfg.ShortCode != "" {
		cfg.BusinessShortCode = cfg.ShortCode
	}

	if strings.TrimSpace(cfg.ConsumerKey) == "" {
		return nil, errors.New("mpesa: consumer key is required")
	}
	if strings.TrimSpace(cfg.ConsumerSecret) == "" {
		return nil, errors.New("mpesa: consumer secret is required")
	}
	if strings.TrimSpace(cfg.Passkey) == "" {
		return nil, errors.New("mpesa: passkey is required")
	}
	if strings.TrimSpace(cfg.BusinessShortCode) == "" {
		return nil, errors.New("mpesa: business short code is required")
	}

	if cfg.BaseURL == "" {
		if cfg.Environment == EnvironmentProduction {
			cfg.BaseURL = "https://api.safaricom.co.ke"
		} else {
			cfg.BaseURL = "https://sandbox.safaricom.co.ke"
		}
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = paykit.NewHTTPClient(paykit.HTTPClientConfig{})
	}

	tokenManager := NewTokenManager(
		cfg.HTTPClient,
		cfg.ConsumerKey,
		cfg.ConsumerSecret,
		cfg.BaseURL,
	)

	client := NewMpesaClient(
		cfg.BaseURL,
		cfg.HTTPClient,
		cfg.Passkey,
		tokenManager,
	)

	return &Gateway{
		client: client,
		cfg:    cfg,
	}, nil
}

// NewWithClient constructs a Gateway using an existing transport Client and Config.
func NewWithClient(client *Client, cfg Config) (*Gateway, error) {
	if client == nil {
		return nil, errors.New("mpesa: client cannot be nil")
	}
	if cfg.BusinessShortCode == "" && cfg.ShortCode != "" {
		cfg.BusinessShortCode = cfg.ShortCode
	}

	return &Gateway{
		client: client,
		cfg:    cfg,
	}, nil
}

// Client returns the underlying raw transport Client.
func (g *Gateway) Client() *Client {
	return g.client
}

// Name returns the canonical provider identifier "mpesa".
func (g *Gateway) Name() string {
	return "mpesa"
}

// Charge initiates a customer payment collection via SIM Toolkit STK Push (Lipa Na M-Pesa Online).
//
// The amount is converted from minor units (cents) to whole Kenyan Shillings. Amounts below
// 1 KES or currencies other than KES are rejected. The phone number is automatically validated
// and normalized to 12-digit E.164 format.
//
// On successful initiation, Charge returns StatusPendingAction along with the CheckoutRequestID,
// indicating that the customer has been prompted on their phone to enter their M-Pesa PIN.
func (g *Gateway) Charge(ctx context.Context, req *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	if req == nil {
		return nil, errors.New("mpesa: charge request cannot be nil")
	}

	if req.Amount.Currency() != "" && req.Amount.Currency() != "KES" {
		return nil, fmt.Errorf("mpesa: unsupported currency %q (only KES is supported)", req.Amount.Currency())
	}
	if req.Amount.Amount() <= 0 {
		return nil, errors.New("mpesa: charge amount must be greater than zero")
	}
	if req.Amount.Amount() < 100 {
		return nil, errors.New("mpesa: charge amount must be at least 1 KES")
	}

	amountKES := int(req.Amount.Amount() / 100)

	phone, err := paykit.FormatKenyanPhone(req.Phone)
	if err != nil {
		return nil, fmt.Errorf("mpesa: invalid phone number: %w", err)
	}

	callbackURL := req.CallbackURL
	if callbackURL == "" {
		callbackURL = g.cfg.CallbackURL
	}
	if callbackURL == "" {
		return nil, errors.New("mpesa: callback URL is required for STK push")
	}

	ref := req.Reference
	if ref == "" {
		ref = "Payment"
	}
	desc := req.Description
	if desc == "" {
		desc = ref
	}

	shortcode := g.cfg.BusinessShortCode
	if req.Metadata != nil && req.Metadata["business_short_code"] != "" {
		shortcode = req.Metadata["business_short_code"]
	}

	trxType := "CustomerPayBillOnline"
	if req.Metadata != nil && req.Metadata["transaction_type"] != "" {
		trxType = req.Metadata["transaction_type"]
	}

	partyB := shortcode
	if req.Metadata != nil && req.Metadata["party_b"] != "" {
		partyB = req.Metadata["party_b"]
	}

	stkReq := STKPushRequest{
		IdempotencyKey:    req.IdempotencyKey,
		BusinessShortCode: shortcode,
		TransactionType:   trxType,
		Amount:            amountKES,
		PartyA:            phone,
		PartyB:            partyB,
		PhoneNumber:       phone,
		CallBackURL:       callbackURL,
		AccountReference:  ref,
		TransactionDesc:   desc,
	}

	stkResp, err := g.client.STKPush(ctx, stkReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: STK push failed: %w", err)
	}

	success := stkResp.ResponseCode == "0"
	status := paykit.StatusPendingAction
	if !success {
		status = paykit.StatusFailed
	}

	message := stkResp.CustomerMessage
	if message == "" {
		message = stkResp.ResponseDescription
	}

	rawJSON, _ := json.Marshal(stkResp)

	return &paykit.ChargeResponse{
		Success:           success,
		Message:           message,
		TransactionID:     stkResp.CheckoutRequestID,
		CheckoutRequestID: stkResp.CheckoutRequestID,
		Status:            status,
		Raw:               rawJSON,
		Metadata: map[string]any{
			"merchantRequestID":   stkResp.MerchantRequestID,
			"responseCode":        stkResp.ResponseCode,
			"responseDescription": stkResp.ResponseDescription,
		},
	}, nil
}

// QueryStatus queries the current state of an initiated STK Push transaction using its CheckoutRequestID.
//
// The CheckoutRequestID must be provided either via req.TransactionID or req.Metadata["checkout_request_id"].
// The response maps Daraja ResultCodes to normalized paykit.TransactionStatus values:
//   - "0": StatusSuccess
//   - "1032": StatusCancelled (customer cancelled on phone)
//   - "1037" / "1025" / "1019": StatusTimeout
//   - Any other non-zero code: StatusFailed
func (g *Gateway) QueryStatus(ctx context.Context, req *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	if req == nil {
		return nil, errors.New("mpesa: status request cannot be nil")
	}

	checkoutID := req.TransactionID
	if checkoutID == "" && req.Metadata != nil {
		checkoutID = req.Metadata["checkout_request_id"]
	}
	if strings.TrimSpace(checkoutID) == "" {
		return nil, errors.New("mpesa: transaction ID or checkout_request_id is required for status query")
	}

	shortcode := g.cfg.BusinessShortCode
	if req.Metadata != nil && req.Metadata["business_short_code"] != "" {
		shortcode = req.Metadata["business_short_code"]
	}

	queryReq := STKPushQueryRequest{
		IdempotencyKey:    req.IdempotencyKey,
		BusinessShortCode: shortcode,
		CheckoutRequestID: checkoutID,
	}

	queryResp, err := g.client.STKPushQuery(ctx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: STK query failed: %w", err)
	}

	var status paykit.TransactionStatus
	switch queryResp.ResultCode {
	case "0":
		status = paykit.StatusSuccess
	case "1032":
		status = paykit.StatusCancelled
	case "1037", "1025", "1019":
		status = paykit.StatusTimeout
	default:
		status = paykit.StatusFailed
	}

	msg := queryResp.ResultDesc
	if msg == "" {
		msg = queryResp.ResponseDescription
	}

	rawJSON, _ := json.Marshal(queryResp)

	return &paykit.StatusResponse{
		Success:       queryResp.ResultCode == "0",
		Message:       msg,
		TransactionID: queryResp.CheckoutRequestID,
		Status:        status,
		Raw:           rawJSON,
		Metadata: map[string]any{
			"merchantRequestID":   queryResp.MerchantRequestID,
			"checkoutRequestID":   queryResp.CheckoutRequestID,
			"resultCode":          queryResp.ResultCode,
			"resultDesc":          queryResp.ResultDesc,
			"responseCode":        queryResp.ResponseCode,
			"responseDescription": queryResp.ResponseDescription,
		},
	}, nil
}

// Refund initiates a full or partial reversal request for a completed M-Pesa transaction.
//
// Initiator credentials and receiver shortcode are loaded from the gateway's Config or
// can be overridden through req.Metadata keys ("initiator", "security_credential", "receiver_party").
func (g *Gateway) Refund(ctx context.Context, req *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	if req == nil {
		return nil, errors.New("mpesa: refund request cannot be nil")
	}
	if strings.TrimSpace(req.TransactionID) == "" {
		return nil, errors.New("mpesa: transaction ID is required for refund")
	}

	amountKES := 0
	if !req.Amount.IsZero() {
		if req.Amount.Currency() != "" && req.Amount.Currency() != "KES" {
			return nil, fmt.Errorf("mpesa: unsupported currency %q (only KES is supported)", req.Amount.Currency())
		}
		if req.Amount.Amount() < 100 {
			return nil, errors.New("mpesa: refund amount must be at least 1 KES")
		}
		amountKES = int(req.Amount.Amount() / 100)
	}

	initiator := g.cfg.InitiatorName
	if req.Metadata != nil && req.Metadata["initiator"] != "" {
		initiator = req.Metadata["initiator"]
	}
	if initiator == "" {
		return nil, errors.New("mpesa: initiator is required for transaction reversal")
	}

	cred := g.cfg.SecurityCredential
	if req.Metadata != nil && req.Metadata["security_credential"] != "" {
		cred = req.Metadata["security_credential"]
	}
	if cred == "" {
		return nil, errors.New("mpesa: security credential is required for transaction reversal")
	}

	receiverParty := g.cfg.BusinessShortCode
	if req.Metadata != nil && req.Metadata["receiver_party"] != "" {
		receiverParty = req.Metadata["receiver_party"]
	}
	if receiverParty == "" {
		return nil, errors.New("mpesa: receiver party (shortcode) is required for transaction reversal")
	}

	resultURL := g.cfg.ResultURL
	if req.Metadata != nil && req.Metadata["result_url"] != "" {
		resultURL = req.Metadata["result_url"]
	}
	timeoutURL := g.cfg.QueueTimeOutURL
	if req.Metadata != nil && req.Metadata["queue_timeout_url"] != "" {
		timeoutURL = req.Metadata["queue_timeout_url"]
	}

	revReq := ReversalRequest{
		IdempotencyKey:     req.IdempotencyKey,
		Initiator:          initiator,
		SecurityCredential: cred,
		TransactionID:      req.TransactionID,
		Amount:             amountKES,
		ReceiverParty:      receiverParty,
		ResultURL:          resultURL,
		QueueTimeOutURL:    timeoutURL,
		Remarks:            req.Reason,
	}

	revResp, err := g.client.ReverseTransaction(ctx, revReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: reversal failed: %w", err)
	}

	success := revResp.ResponseCode == "0"
	status := paykit.StatusProcessing
	if !success {
		status = paykit.StatusFailed
	}

	rawJSON, _ := json.Marshal(revResp)

	return &paykit.RefundResponse{
		Success:       success,
		Message:       revResp.ResponseDescription,
		RefundID:      revResp.ConversationID,
		TransactionID: req.TransactionID,
		Status:        status,
		Raw:           rawJSON,
		Metadata: map[string]any{
			"conversationID":           revResp.ConversationID,
			"originatorConversationID": revResp.OriginatorConversationID,
			"responseCode":             revResp.ResponseCode,
		},
	}, nil
}

// Disburse transfers funds from the organization business account to a recipient customer account (B2C).
//
// The recipient phone number is normalized to 12-digit E.164 format and the disbursement amount
// is converted to whole Kenyan Shillings. B2C operations are processed asynchronously by Daraja;
// a successful initiation returns StatusProcessing along with the ConversationID.
func (g *Gateway) Disburse(ctx context.Context, req *paykit.DisbursementRequest) (*paykit.DisbursementResponse, error) {
	if req == nil {
		return nil, errors.New("mpesa: disbursement request cannot be nil")
	}

	if req.Amount.Currency() != "" && req.Amount.Currency() != "KES" {
		return nil, fmt.Errorf("mpesa: unsupported currency %q (only KES is supported)", req.Amount.Currency())
	}
	if req.Amount.Amount() <= 0 {
		return nil, errors.New("mpesa: disbursement amount must be greater than zero")
	}
	if req.Amount.Amount() < 100 {
		return nil, errors.New("mpesa: disbursement amount must be at least 1 KES")
	}

	amountKES := int(req.Amount.Amount() / 100)

	phone, err := paykit.FormatKenyanPhone(req.Phone)
	if err != nil {
		return nil, fmt.Errorf("mpesa: invalid recipient phone: %w", err)
	}

	initiator := g.cfg.InitiatorName
	if req.Metadata != nil && req.Metadata["initiator_name"] != "" {
		initiator = req.Metadata["initiator_name"]
	}
	if initiator == "" {
		return nil, errors.New("mpesa: initiator name is required for disbursement")
	}

	cred := g.cfg.SecurityCredential
	if req.Metadata != nil && req.Metadata["security_credential"] != "" {
		cred = req.Metadata["security_credential"]
	}
	if cred == "" {
		return nil, errors.New("mpesa: security credential is required for disbursement")
	}

	partyA := g.cfg.BusinessShortCode
	if req.Metadata != nil && req.Metadata["party_a"] != "" {
		partyA = req.Metadata["party_a"]
	}
	if partyA == "" {
		return nil, errors.New("mpesa: party A short code is required for disbursement")
	}

	resultURL := req.CallbackURL
	if resultURL == "" {
		resultURL = g.cfg.ResultURL
	}
	timeoutURL := g.cfg.QueueTimeOutURL
	if req.Metadata != nil && req.Metadata["queue_timeout_url"] != "" {
		timeoutURL = req.Metadata["queue_timeout_url"]
	}

	commandID := "BusinessPayment"
	if req.Metadata != nil && req.Metadata["command_id"] != "" {
		commandID = req.Metadata["command_id"]
	}

	remarks := req.Description
	if remarks == "" {
		remarks = req.Reference
	}
	if remarks == "" {
		remarks = "Disbursement"
	}

	b2cReq := B2CRequest{
		IdempotencyKey:     req.IdempotencyKey,
		InitiatorName:      initiator,
		SecurityCredential: cred,
		CommandID:          commandID,
		Amount:             amountKES,
		PartyA:             partyA,
		PartyB:             phone,
		Remarks:            remarks,
		QueueTimeOutURL:    timeoutURL,
		ResultURL:          resultURL,
	}

	b2cResp, err := g.client.B2CPayment(ctx, b2cReq)
	if err != nil {
		return nil, fmt.Errorf("mpesa: disbursement failed: %w", err)
	}

	success := b2cResp.ResponseCode == "0"
	status := paykit.StatusProcessing
	if !success {
		status = paykit.StatusFailed
	}

	rawJSON, _ := json.Marshal(b2cResp)

	return &paykit.DisbursementResponse{
		Success:       success,
		Message:       b2cResp.ResponseDescription,
		TransactionID: b2cResp.ConversationID,
		Status:        status,
		Raw:           rawJSON,
		Metadata: map[string]any{
			"conversationID":           b2cResp.ConversationID,
			"originatorConversationID": b2cResp.OriginatorConversationID,
			"responseCode":             b2cResp.ResponseCode,
		},
	}, nil
}
