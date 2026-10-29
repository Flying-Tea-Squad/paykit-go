package paykit

import (
	"encoding/json"
)

// ChargeResponse represents the result of a charge initiation.
type ChargeResponse struct {
	Success           bool              `json:"success"`
	Message           string            `json:"message,omitempty"`
	TransactionID     string            `json:"transaction_id,omitempty"`
	CheckoutRequestID string            `json:"checkout_request_id,omitempty"`
	Status            TransactionStatus `json:"status,omitempty"`
	Raw               json.RawMessage   `json:"raw,omitempty"`
	Metadata          map[string]any    `json:"metadata,omitempty"`
}

// DisbursementResponse represents the result of a disbursement initiation.
type DisbursementResponse struct {
	Success       bool              `json:"success"`
	Message       string            `json:"message,omitempty"`
	TransactionID string            `json:"transaction_id,omitempty"`
	Status        TransactionStatus `json:"status,omitempty"`
	Raw           json.RawMessage   `json:"raw,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

// StatusResponse represents the result of a transaction status query.
type StatusResponse struct {
	Success       bool              `json:"success"`
	Message       string            `json:"message,omitempty"`
	TransactionID string            `json:"transaction_id,omitempty"`
	Status        TransactionStatus `json:"status,omitempty"`
	Amount        Money             `json:"amount,omitempty"`
	Raw           json.RawMessage   `json:"raw,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

// RefundResponse represents the result of a refund initiation.
type RefundResponse struct {
	Success       bool              `json:"success"`
	Message       string            `json:"message,omitempty"`
	RefundID      string            `json:"refund_id,omitempty"`
	TransactionID string            `json:"transaction_id,omitempty"`
	Status        TransactionStatus `json:"status,omitempty"`
	Raw           json.RawMessage   `json:"raw,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

// BalanceResponse contains account balance information.
type BalanceResponse struct {
	Success          bool            `json:"success"`
	Message          string          `json:"message,omitempty"`
	LedgerBalance    Money           `json:"ledger_balance"`
	AvailableBalance Money           `json:"available_balance,omitempty"`
	Raw              json.RawMessage `json:"raw,omitempty"`
	Metadata         map[string]any  `json:"metadata,omitempty"`
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
