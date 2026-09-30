package paykit

import "encoding/json"

// Response represents the standardized response returned by all payment providers.
// Deprecated: Use ChargeResponse, DisbursementResponse, StatusResponse, or RefundResponse instead.
// Kept for backward compatibility with the legacy Gateway interface.
type Response struct {
	// Success indicates whether the operation completed successfully.
	Success bool
	// Message contains a human-readable description of the result.
	Message string
	// TransactionID is the provider's transaction identifier.
	TransactionID string
	// Authorization contains any authorization reference returned by the provider.
	Authorization string
	// Raw contains the original provider response for debugging purposes.
	Raw json.RawMessage
	// ErrorCode contains a standardized or provider-specific error code.
	ErrorCode string
	// Metadata stores additional provider-specific information.
	Metadata map[string]any
}

// ChargeResponse represents the result of a charge initiation.
type ChargeResponse struct {
	// Success indicates whether the charge was initiated successfully.
	Success bool
	// TransactionID is the provider's transaction identifier.
	TransactionID string
	// CheckoutRequestID is the provider-specific checkout reference (e.g., M-Pesa CheckoutRequestID).
	CheckoutRequestID string
	// Status is the current transaction status.
	Status TransactionStatus
	// Message is a human-readable description of the result.
	Message string
	// Raw contains the original provider response for debugging.
	Raw json.RawMessage
}

// DisbursementResponse represents the result of a disbursement initiation.
type DisbursementResponse struct {
	// Success indicates whether the disbursement was initiated successfully.
	Success bool
	// TransactionID is the provider's transaction identifier.
	TransactionID string
	// Status is the current transaction status.
	Status TransactionStatus
	// Message is a human-readable description of the result.
	Message string
	// Raw contains the original provider response for debugging.
	Raw json.RawMessage
}

// StatusResponse represents the result of a transaction status query.
type StatusResponse struct {
	// TransactionID is the provider's transaction identifier.
	TransactionID string
	// Status is the current transaction status.
	Status TransactionStatus
	// Amount is the actual settled amount (may differ from requested).
	Amount Money
	// Message is a human-readable description of the result.
	Message string
	// Raw contains the original provider response for debugging.
	Raw json.RawMessage
}

// RefundResponse represents the result of a refund initiation.
type RefundResponse struct {
	// Success indicates whether the refund was initiated successfully.
	Success bool
	// TransactionID is the provider's transaction identifier for the refund.
	TransactionID string
	// Status is the current refund status.
	Status TransactionStatus
	// Message is a human-readable description of the result.
	Message string
	// Raw contains the original provider response for debugging.
	Raw json.RawMessage
}
