package paykit

import (
	"encoding/json"
	"time"
)

// ChargeResponse represents the outcome of a Charge operation.
type ChargeResponse struct {
	// Success indicates whether the charge was accepted or processed successfully.
	Success bool `json:"success"`

	// Message contains human-readable status information or provider error descriptions.
	Message string `json:"message,omitempty"`

	// TransactionID is the provider's unique transaction identifier.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status indicates normalized or provider state (e.g. "pending", "completed", "failed").
	Status string `json:"status,omitempty"`

	// Raw holds the original raw provider payload for auditing and troubleshooting.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds extra provider-specific properties (e.g. checkout request ID, redirect URLs).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DisbursementResponse represents the outcome of a Disburse operation.
type DisbursementResponse struct {
	// Success indicates whether the payout was accepted or processed successfully.
	Success bool `json:"success"`

	// Message contains human-readable status information.
	Message string `json:"message,omitempty"`

	// TransactionID is the provider's payout transaction reference.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status indicates transaction state (e.g. "pending", "completed", "failed").
	Status string `json:"status,omitempty"`

	// Raw contains the unparsed provider response.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds provider-specific payout attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StatusResponse represents the result of a QueryStatus operation.
type StatusResponse struct {
	// Success indicates whether the status query itself completed successfully.
	Success bool `json:"success"`

	// Message is a descriptive status message from the provider.
	Message string `json:"message,omitempty"`

	// TransactionID is the provider's transaction identifier.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status is the normalized transaction state.
	Status string `json:"status,omitempty"`

	// Raw contains the complete unparsed provider response.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds auxiliary status details.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RefundResponse represents the result of a Refund operation.
type RefundResponse struct {
	// Success indicates whether the reversal was initiated or completed successfully.
	Success bool `json:"success"`

	// Message provides human-readable feedback.
	Message string `json:"message,omitempty"`

	// RefundID is the provider's distinct refund or reversal identifier.
	RefundID string `json:"refund_id,omitempty"`

	// TransactionID is the original transaction reference being refunded.
	TransactionID string `json:"transaction_id,omitempty"`

	// Status reflects the refund state.
	Status string `json:"status,omitempty"`

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

	// Currency is the ISO 4217 currency code for the balance.
	Currency string `json:"currency,omitempty"`

	// Balance represents the total ledger balance in minor units.
	Balance int `json:"balance"`

	// Available represents the immediately usable funds in minor units.
	Available int `json:"available,omitempty"`

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

	// Amount is the transferred sum in minor currency units.
	Amount int `json:"amount,omitempty"`

	// Currency is the ISO 4217 currency code.
	Currency string `json:"currency,omitempty"`

	// Phone is the customer or recipient MSISDN in E.164.
	Phone string `json:"phone,omitempty"`

	// Timestamp is the recorded settlement time.
	Timestamp time.Time `json:"timestamp,omitempty"`

	// Raw contains the unparsed incoming callback body for auditing.
	Raw []byte `json:"raw,omitempty"`

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
