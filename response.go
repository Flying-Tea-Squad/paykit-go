package paykit

import (
	"encoding/json"
	"time"
)

// ChargeResponse represents the result of a charge initiation.
type ChargeResponse struct {
	// Success indicates whether the charge was initiated or processed successfully.
	Success bool `json:"success"`

	// Message contains human-readable status information or provider error descriptions.
	Message string `json:"message,omitempty"`

	// TransactionID is the provider's unique transaction identifier.
	TransactionID string `json:"transaction_id,omitempty"`

	// CheckoutRequestID is the provider-specific checkout reference (e.g., M-Pesa CheckoutRequestID).
	CheckoutRequestID string `json:"checkout_request_id,omitempty"`

	// Status indicates the current normalized transaction status.
	Status TransactionStatus `json:"status,omitempty"`

	// Raw holds the original raw provider payload for auditing and troubleshooting.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds extra provider-specific properties (e.g. checkout request ID, redirect URLs).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DisbursementResponse represents the result of a disbursement initiation.
type DisbursementResponse struct {
	// Success indicates whether the payout was accepted or processed successfully.
	Success bool `json:"success"`

	// Message contains human-readable status information.
	Message string `json:"message,omitempty"`

	// TransactionID is the provider's payout transaction reference.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status indicates the current transaction status.
	Status TransactionStatus `json:"status,omitempty"`

	// Raw contains the unparsed provider response.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds provider-specific payout attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StatusResponse represents the result of a transaction status query.
type StatusResponse struct {
	// Success indicates whether the status query itself completed successfully.
	Success bool `json:"success"`

	// Message is a descriptive status message from the provider.
	Message string `json:"message,omitempty"`

	// TransactionID is the provider's transaction identifier.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status is the normalized transaction state.
	Status TransactionStatus `json:"status,omitempty"`

	// Amount is the actual settled amount (may differ from requested).
	Amount Money `json:"amount,omitempty"`

	// Raw contains the complete unparsed provider response.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds auxiliary status details.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RefundResponse represents the result of a refund initiation.
type RefundResponse struct {
	// Success indicates whether the refund/reversal was initiated successfully.
	Success bool `json:"success"`

	// Message provides human-readable feedback.
	Message string `json:"message,omitempty"`

	// RefundID is the provider's distinct refund or reversal identifier.
	RefundID string `json:"refund_id,omitempty"`

	// TransactionID is the original transaction reference being refunded.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status reflects the refund status.
	Status TransactionStatus `json:"status,omitempty"`

	// Raw contains the provider's raw refund payload.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds auxiliary refund attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// BalanceResponse contains account balance information.
type BalanceResponse struct {
	// Success indicates whether balance retrieval succeeded.
	Success bool `json:"success"`

	// Message provides human-readable feedback.
	Message string `json:"message,omitempty"`

	// LedgerBalance represents the total ledger balance.
	LedgerBalance Money `json:"ledger_balance"`

	// AvailableBalance represents the immediately usable funds.
	AvailableBalance Money `json:"available_balance,omitempty"`

	// Raw contains the provider's raw balance payload.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds auxiliary balance properties.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Event represents a normalized incoming asynchronous notification from a payment provider
// (such as an STK Push callback, B2C result notification, or Pesapal IPN).
type Event struct {
	// ID is the unique event delivery identifier (if provided by the webhook source).
	ID string `json:"id,omitempty"`

	// Provider identifies the originating payment gateway (e.g. "mpesa", "airtel", "pesapal").
	Provider string `json:"provider"`

	// Type specifies the event category (e.g. "charge.completed", "disbursement.failed").
	Type string `json:"type"`

	// TransactionID is the provider's unique settlement transaction identifier.
	TransactionID string `json:"transaction_id,omitempty"`

	// Reference is the merchant's tracking or account reference.
	Reference string `json:"reference,omitempty"`

	// Status is the normalized transaction status.
	Status TransactionStatus `json:"status,omitempty"`

	// Amount is the transferred sum as Money.
	Amount Money `json:"amount,omitempty"`

	// Phone is the customer or recipient MSISDN in E.164.
	Phone string `json:"phone,omitempty"`

	// Timestamp is the recorded settlement time.
	Timestamp time.Time `json:"timestamp,omitempty"`

	// Raw contains the unparsed incoming callback body for auditing.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds provider-specific webhook fields.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Response represents the legacy standardized response.
// Deprecated: Prefer operation-specific responses such as ChargeResponse or DisbursementResponse.
type Response struct {
	Success       bool            `json:"success"`
	Message       string          `json:"message,omitempty"`
	TransactionID string          `json:"transaction_id,omitempty"`
	Authorization string          `json:"authorization,omitempty"`
	Raw           json.RawMessage `json:"raw,omitempty"`
	ErrorCode     string          `json:"error_code,omitempty"`
	Metadata      map[string]any  `json:"metadata,omitempty"`
}
