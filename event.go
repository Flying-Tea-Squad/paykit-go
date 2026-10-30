package paykit

import (
	"encoding/json"
	"time"
)

// EventType represents the normalized category of an asynchronous payment event.
type EventType string

const (
	// EventChargeCompleted indicates a customer payment collection settled successfully.
	EventChargeCompleted EventType = "charge.completed"

	// EventChargeFailed indicates a customer payment collection failed or was declined.
	EventChargeFailed EventType = "charge.failed"

	// EventDisbursementCompleted indicates a B2C payout settled successfully.
	EventDisbursementCompleted EventType = "disbursement.completed"

	// EventDisbursementFailed indicates a B2C payout failed or was rejected.
	EventDisbursementFailed EventType = "disbursement.failed"

	// EventC2BReceived indicates an inbound customer-to-business payment was received,
	// such as an M-Pesa Paybill or Till Number confirmation.
	EventC2BReceived EventType = "c2b.received"
)

// Event represents a normalized, provider-agnostic asynchronous payment notification
// produced by implementations of the WebhookHandler interface.
type Event struct {
	// ID is the unique event delivery identifier, when supplied by the provider.
	ID string `json:"id,omitempty"`

	// Provider identifies the originating payment gateway (e.g. "mpesa", "airtel", "pesapal").
	Provider string `json:"provider"`

	// Type specifies the normalized event category.
	Type EventType `json:"type"`

	// TransactionID is the provider's unique settlement transaction identifier.
	TransactionID string `json:"transaction_id,omitempty"`

	// Reference is the merchant's tracking or account reference for the transaction.
	Reference string `json:"reference,omitempty"`

	// Amount is the transferred sum expressed in minor units with an ISO-4217 currency.
	Amount Money `json:"amount"`

	// Phone is the customer or recipient MSISDN normalized to E.164.
	Phone string `json:"phone,omitempty"`

	// Timestamp is the recorded settlement time reported by the provider.
	Timestamp time.Time `json:"timestamp"`

	// Status is the normalized transaction state associated with this event.
	Status TransactionStatus `json:"status,omitempty"`

	// Raw holds the unparsed incoming callback body for auditing and signature replay.
	Raw json.RawMessage `json:"raw,omitempty"`

	// Metadata holds provider-specific fields that have no normalized equivalent.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// IsSuccess reports whether the event represents a successful terminal state.
// It is nil-safe and returns false when called on a nil receiver.
func (e *Event) IsSuccess() bool {
	if e == nil {
		return false
	}
	return e.Status == StatusSuccess
}

// IsFailed reports whether the event represents a failed terminal state.
// It is nil-safe and returns false when called on a nil receiver.
func (e *Event) IsFailed() bool {
	if e == nil {
		return false
	}
	return e.Status == StatusFailed
}

// IsCharge reports whether the event relates to a customer payment collection.
// It is nil-safe and returns false when called on a nil receiver.
func (e *Event) IsCharge() bool {
	if e == nil {
		return false
	}
	return e.Type == EventChargeCompleted || e.Type == EventChargeFailed
}

// IsDisbursement reports whether the event relates to a B2C payout.
// It is nil-safe and returns false when called on a nil receiver.
func (e *Event) IsDisbursement() bool {
	if e == nil {
		return false
	}
	return e.Type == EventDisbursementCompleted || e.Type == EventDisbursementFailed
}
