package paykit

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Money represents an immutable monetary amount in minor units (cents) with an ISO-4217 currency code.
//
// By storing amounts in integer minor units (e.g., 10050 for KES 100.50), Money avoids
// floating-point precision loss and rounding issues common in financial computations.
//
// Money values are immutable. Arithmetic methods ([Money.Add], [Money.Sub], [Money.Mul], [Money.Cmp])
// return new Money instances and enforce currency matching.
type Money struct {
	amount   int64
	currency string
}

// NewMoney creates a Money value from minor units (cents) and an ISO-4217 currency code.
// The currency code is automatically trimmed and converted to uppercase.
//
// Example:
//
//	m := paykit.NewMoney(10050, "KES") // KES 100.50
func NewMoney(minorUnits int64, currency string) Money {
	return Money{
		amount:   minorUnits,
		currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

// NewKES creates a Money value from Kenyan shillings in major units (e.g., 100.50).
// The input is converted to cents (minor units) by multiplying by 100 and rounding.
//
// For example, NewKES(100.50) creates a Money value representing 10050 KES cents.
// If shillings is NaN or Inf, a zero-value KES Money is returned.
func NewKES(shillings float64) Money {
	if math.IsNaN(shillings) || math.IsInf(shillings, 0) {
		return Money{amount: 0, currency: "KES"}
	}
	minorUnits := int64(math.Round(shillings * 100))
	return Money{amount: minorUnits, currency: "KES"}
}

// Amount returns the amount in integer minor units (cents).
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the uppercase ISO-4217 currency code (e.g., "KES", "USD").
func (m Money) Currency() string {
	return m.currency
}

// IsZero reports whether the monetary amount is zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// String returns a human-readable representation in major units with the currency code.
//
// Example:
//
//	paykit.NewKES(100.50).String() // "KES 100.50"
func (m Money) String() string {
	major := float64(m.amount) / 100
	return fmt.Sprintf("%s %.2f", m.currency, major)
}

// Float64 returns the amount in major units as a float64.
//
// Caution: Float64 should only be used for display or legacy integrations.
// Prefer [Money.Amount] for precise financial calculations and persistence.
func (m Money) Float64() float64 {
	return float64(m.amount) / 100
}

// Add returns the sum of m and other.
// Both operands must have the same currency; otherwise Add panics.
func (m Money) Add(other Money) Money {
	if m.currency != other.currency {
		panic(fmt.Sprintf("currency mismatch: %s != %s", m.currency, other.currency))
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}
}

// Sub returns the difference m - other.
// Both operands must have the same currency; otherwise Sub panics.
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
// Returns -1 if m < other, 0 if m == other, and 1 if m > other.
// Both operands must have the same currency; otherwise Cmp panics.
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

// MarshalJSON implements json.Marshaler, serializing Money as an object with amount and currency.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}{m.amount, m.currency})
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

// TransactionStatus represents the standardized lifecycle state of a payment transaction.
//
// Lifecycle transitions typically proceed from an in-flight state ([StatusPendingAction] or
// [StatusProcessing]) to a terminal state ([StatusSuccess], [StatusFailed], [StatusCancelled],
// or [StatusTimeout]). Use [TransactionStatus.IsTerminal] to verify whether a transaction has finished.
type TransactionStatus string

const (
	// StatusPendingAction indicates the transaction awaits customer action (e.g., entering PIN on an STK Push).
	// This is a non-terminal state.
	StatusPendingAction TransactionStatus = "pending_action"

	// StatusProcessing indicates the transaction is in flight and actively being processed by the provider.
	// This is a non-terminal state.
	StatusProcessing TransactionStatus = "processing"

	// StatusSuccess indicates the transaction completed and settled successfully.
	// This is a terminal state.
	StatusSuccess TransactionStatus = "success"

	// StatusFailed indicates the transaction failed (e.g., insufficient funds, decline, invalid account).
	// This is a terminal state.
	StatusFailed TransactionStatus = "failed"

	// StatusCancelled indicates the transaction was cancelled by the customer or merchant.
	// This is a terminal state.
	StatusCancelled TransactionStatus = "cancelled"

	// StatusTimeout indicates the transaction timed out waiting for customer interaction or provider response.
	// This is a terminal state.
	StatusTimeout TransactionStatus = "timeout"

	// StatusUnknown indicates an unrecognized or unmapped provider status.
	// It is treated as a safe non-terminal default.
	StatusUnknown TransactionStatus = "unknown"
)

// IsTerminal returns true if the status is a final, immutable state where no further
// status transitions are expected ([StatusSuccess], [StatusFailed], [StatusCancelled], or [StatusTimeout]).
func (s TransactionStatus) IsTerminal() bool {
	switch s {
	case StatusSuccess, StatusFailed, StatusCancelled, StatusTimeout:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer, returning the string value of the transaction status.
func (s TransactionStatus) String() string {
	return string(s)
}

// ParseTransactionStatus parses an arbitrary status string into a normalized [TransactionStatus].
// Returns [StatusUnknown] for unrecognized values (a safe non-terminal default).
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
