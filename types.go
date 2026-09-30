package paykit

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Money represents a monetary amount in minor units (cents) with an ISO-4217 currency code.
// It is an immutable value object.
type Money struct {
	amount   int64
	currency string
}

// NewMoney creates a Money value from minor units and a currency code.
// The currency should be a valid ISO-4217 code (e.g., "KES", "USD").
func NewMoney(minorUnits int64, currency string) Money {
	return Money{
		amount:   minorUnits,
		currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

// NewKES creates a Money value from Kenyan shillings (major units).
// The input is converted to cents (minor units) by multiplying by 100.
// For example, NewKES(100.50) creates Money{10050, "KES"}.
func NewKES(shillings float64) Money {
	if math.IsNaN(shillings) || math.IsInf(shillings, 0) {
		return Money{amount: 0, currency: "KES"}
	}
	minorUnits := int64(math.Round(shillings * 100))
	return Money{amount: minorUnits, currency: "KES"}
}

// Amount returns the amount in minor units (cents).
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the ISO-4217 currency code.
func (m Money) Currency() string {
	return m.currency
}

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// String returns a human-readable representation in major units with currency code.
// Example: "KES 100.50"
func (m Money) String() string {
	major := float64(m.amount) / 100
	return fmt.Sprintf("%s %.2f", m.currency, major)
}

// Float64 returns the amount in major units as a float64.
// Use with caution for financial calculations; prefer Amount() for precision.
func (m Money) Float64() float64 {
	return float64(m.amount) / 100
}

// Add returns the sum of m and other.
// Both must have the same currency; otherwise it panics.
func (m Money) Add(other Money) Money {
	if m.currency != other.currency {
		panic(fmt.Sprintf("currency mismatch: %s != %s", m.currency, other.currency))
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}
}

// Sub returns the difference m - other.
// Both must have the same currency; otherwise it panics.
func (m Money) Sub(other Money) Money {
	if m.currency != other.currency {
		panic(fmt.Sprintf("currency mismatch: %s != %s", m.currency, other.currency))
	}
	return Money{amount: m.amount - other.amount, currency: m.currency}
}

// Mul returns m multiplied by a scalar factor.
func (m Money) Mul(factor int64) Money {
	return Money{amount: m.amount * factor, currency: m.currency}
}

// Cmp compares m with other.
// Returns -1 if m < other, 0 if m == other, 1 if m > other.
// Both must have the same currency; otherwise it panics.
func (m Money) Cmp(other Money) int {
	if m.currency != other.currency {
		panic(fmt.Sprintf("currency mismatch: %s != %s", m.currency, other.currency))
	}
	if m.amount < other.amount {
		return -1
	}
	if m.amount > other.amount {
		return 1
	}
	return 0
}

// MarshalJSON implements json.Marshaler.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"amount":%d,"currency":"%s"}`, m.amount, m.currency)), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *Money) UnmarshalJSON(data []byte) error {
	var raw struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.amount = raw.Amount
	m.currency = strings.ToUpper(strings.TrimSpace(raw.Currency))
	return nil
}

// TransactionStatus represents the standardized state of a payment transaction.
type TransactionStatus string

const (
	// StatusPendingAction means the transaction awaits user action (e.g., entering PIN on STK Push).
	StatusPendingAction TransactionStatus = "pending_action"
	// StatusProcessing means the transaction is in flight, being processed by the provider.
	StatusProcessing TransactionStatus = "processing"
	// StatusSuccess means the transaction completed successfully.
	StatusSuccess TransactionStatus = "success"
	// StatusFailed means the transaction failed (insufficient funds, declined, etc.).
	StatusFailed TransactionStatus = "failed"
	// StatusCancelled means the transaction was cancelled by the user or merchant.
	StatusCancelled TransactionStatus = "cancelled"
	// StatusTimeout means the transaction timed out waiting for user action or provider response.
	StatusTimeout TransactionStatus = "timeout"
	// StatusUnknown means the transaction state is not recognized.
	// Used as a safe default for unrecognized provider statuses (non-terminal).
	StatusUnknown TransactionStatus = "unknown"
)

// IsTerminal returns true if the status is a final state (no further transitions expected).
func (s TransactionStatus) IsTerminal() bool {
	switch s {
	case StatusSuccess, StatusFailed, StatusCancelled, StatusTimeout:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (s TransactionStatus) String() string {
	return string(s)
}

// ParseTransactionStatus parses a string into a TransactionStatus.
// Returns StatusUnknown for unrecognized values (non-terminal safe default).
func ParseTransactionStatus(s string) TransactionStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending_action", "pending":
		return StatusPendingAction
	case "processing", "in_progress":
		return StatusProcessing
	case "success", "completed", "complete":
		return StatusSuccess
	case "failed", "failure", "declined":
		return StatusFailed
	case "cancelled", "canceled", "voided":
		return StatusCancelled
	case "timeout", "timed_out":
		return StatusTimeout
	default:
		return StatusUnknown
	}
}
