package paykit

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name         string
		minorUnits   int64
		currency     string
		wantAmount   int64
		wantCurrency string
	}{
		{"positive KES", 10050, "KES", 10050, "KES"},
		{"zero amount", 0, "USD", 0, "USD"},
		{"negative amount", -500, "EUR", -500, "EUR"},
		{"lowercase currency", 1000, "kes", 1000, "KES"},
		{"currency with spaces", 2000, "  usd  ", 2000, "USD"},
		{"mixed case currency", 3000, "UsD", 3000, "USD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMoney(tt.minorUnits, tt.currency)
			assert.Equal(t, tt.wantAmount, m.Amount())
			assert.Equal(t, tt.wantCurrency, m.Currency())
		})
	}
}

func TestNewKES(t *testing.T) {
	tests := []struct {
		name       string
		shillings  float64
		wantAmount int64
	}{
		{"whole shillings", 100.00, 10000},
		{"with cents", 100.50, 10050},
		{"fractional cents rounded", 100.005, 10001}, // rounds to nearest cent
		{"fractional cents rounded down", 100.004, 10000},
		{"zero", 0.0, 0},
		{"negative", -50.25, -5025},
		{"large amount", 1000000.99, 100000099},
		{"NaN returns zero", math.NaN(), 0},
		{"positive infinity", math.Inf(1), 0},
		{"negative infinity", math.Inf(-1), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewKES(tt.shillings)
			assert.Equal(t, tt.wantAmount, m.Amount())
			assert.Equal(t, "KES", m.Currency())
		})
	}
}

func TestMoney_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		m        Money
		expected bool
	}{
		{"zero amount", NewMoney(0, "KES"), true},
		{"positive amount", NewMoney(1, "KES"), false},
		{"negative amount", NewMoney(-1, "KES"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.m.IsZero())
		})
	}
}

func TestMoney_String(t *testing.T) {
	tests := []struct {
		name     string
		m        Money
		expected string
	}{
		{"KES whole", NewMoney(10000, "KES"), "KES 100.00"},
		{"KES with cents", NewMoney(10050, "KES"), "KES 100.50"},
		{"USD", NewMoney(5000, "USD"), "USD 50.00"},
		{"zero", NewMoney(0, "KES"), "KES 0.00"},
		{"negative", NewMoney(-2500, "EUR"), "EUR -25.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.m.String())
		})
	}
}

func TestMoney_Float64(t *testing.T) {
	tests := []struct {
		name     string
		m        Money
		expected float64
	}{
		{"whole", NewMoney(10000, "KES"), 100.0},
		{"with cents", NewMoney(10050, "KES"), 100.5},
		{"zero", NewMoney(0, "KES"), 0.0},
		{"negative", NewMoney(-5000, "USD"), -50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.m.Float64())
		})
	}
}

func TestMoney_Add(t *testing.T) {
	tests := []struct {
		name       string
		a          Money
		b          Money
		wantAmount int64
		wantPanic  bool
	}{
		{"same currency", NewMoney(10000, "KES"), NewMoney(5000, "KES"), 15000, false},
		{"zero addend", NewMoney(10000, "KES"), NewMoney(0, "KES"), 10000, false},
		{"negative addend", NewMoney(10000, "KES"), NewMoney(-3000, "KES"), 7000, false},
		{"different currency panics", NewMoney(10000, "KES"), NewMoney(5000, "USD"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() { tt.a.Add(tt.b) })
			} else {
				result := tt.a.Add(tt.b)
				assert.Equal(t, tt.wantAmount, result.Amount())
				assert.Equal(t, "KES", result.Currency())
			}
		})
	}
}

func TestMoney_Sub(t *testing.T) {
	tests := []struct {
		name       string
		a          Money
		b          Money
		wantAmount int64
		wantPanic  bool
	}{
		{"same currency", NewMoney(10000, "KES"), NewMoney(3000, "KES"), 7000, false},
		{"zero subtrahend", NewMoney(10000, "KES"), NewMoney(0, "KES"), 10000, false},
		{"negative result", NewMoney(3000, "KES"), NewMoney(10000, "KES"), -7000, false},
		{"different currency panics", NewMoney(10000, "KES"), NewMoney(5000, "USD"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() { tt.a.Sub(tt.b) })
			} else {
				result := tt.a.Sub(tt.b)
				assert.Equal(t, tt.wantAmount, result.Amount())
				assert.Equal(t, "KES", result.Currency())
			}
		})
	}
}

func TestMoney_Mul(t *testing.T) {
	tests := []struct {
		name       string
		m          Money
		factor     int64
		wantAmount int64
	}{
		{"multiply by 2", NewMoney(5000, "KES"), 2, 10000},
		{"multiply by 0", NewMoney(5000, "KES"), 0, 0},
		{"multiply by negative", NewMoney(5000, "KES"), -3, -15000},
		{"zero amount", NewMoney(0, "KES"), 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.m.Mul(tt.factor)
			assert.Equal(t, tt.wantAmount, result.Amount())
			assert.Equal(t, "KES", result.Currency())
		})
	}
}

func TestMoney_Cmp(t *testing.T) {
	tests := []struct {
		name      string
		a         Money
		b         Money
		want      int
		wantPanic bool
	}{
		{"less than", NewMoney(5000, "KES"), NewMoney(10000, "KES"), -1, false},
		{"equal", NewMoney(5000, "KES"), NewMoney(5000, "KES"), 0, false},
		{"greater than", NewMoney(10000, "KES"), NewMoney(5000, "KES"), 1, false},
		{"negative vs positive", NewMoney(-5000, "KES"), NewMoney(5000, "KES"), -1, false},
		{"different currency panics", NewMoney(5000, "KES"), NewMoney(5000, "USD"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() { tt.a.Cmp(tt.b) })
			} else {
				assert.Equal(t, tt.want, tt.a.Cmp(tt.b))
			}
		})
	}
}

func TestTransactionStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   TransactionStatus
		expected bool
	}{
		{"pending_action not terminal", StatusPendingAction, false},
		{"processing not terminal", StatusProcessing, false},
		{"success terminal", StatusSuccess, true},
		{"failed terminal", StatusFailed, true},
		{"cancelled terminal", StatusCancelled, true},
		{"timeout terminal", StatusTimeout, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.IsTerminal())
		})
	}
}

func TestTransactionStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   TransactionStatus
		expected string
	}{
		{"pending_action", StatusPendingAction, "pending_action"},
		{"processing", StatusProcessing, "processing"},
		{"success", StatusSuccess, "success"},
		{"failed", StatusFailed, "failed"},
		{"cancelled", StatusCancelled, "cancelled"},
		{"timeout", StatusTimeout, "timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestParseTransactionStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TransactionStatus
	}{
		{"pending_action exact", "pending_action", StatusPendingAction},
		{"pending alias", "pending", StatusPendingAction},
		{"processing exact", "processing", StatusProcessing},
		{"in_progress alias", "in_progress", StatusProcessing},
		{"success exact", "success", StatusSuccess},
		{"completed alias", "completed", StatusSuccess},
		{"complete alias", "complete", StatusSuccess},
		{"failed exact", "failed", StatusFailed},
		{"failure alias", "failure", StatusFailed},
		{"declined alias", "declined", StatusFailed},
		{"cancelled exact", "cancelled", StatusCancelled},
		{"canceled US spelling", "canceled", StatusCancelled},
		{"voided alias", "voided", StatusCancelled},
		{"timeout exact", "timeout", StatusTimeout},
		{"timed_out alias", "timed_out", StatusTimeout},
		{"case insensitive", "SUCCESS", StatusSuccess},
		{"with whitespace", "  pending  ", StatusPendingAction},
		{"unknown defaults to failed", "unknown_status", StatusFailed},
		{"empty defaults to failed", "", StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseTransactionStatus(tt.input))
		})
	}
}

func TestMoney_MarshalJSON(t *testing.T) {
	m := NewMoney(10050, "KES")
	data, err := m.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"amount":10050,"currency":"KES"}`, string(data))
}

func TestMoney_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAmount int64
		wantCurr   string
		wantErr    bool
	}{
		{"valid", `{"amount":10050,"currency":"KES"}`, 10050, "KES", false},
		{"lowercase currency", `{"amount":5000,"currency":"usd"}`, 5000, "USD", false},
		{"currency with spaces", `{"amount":2000,"currency":"  eur  "}`, 2000, "EUR", false},
		{"invalid json", `not json`, 0, "", true},
		{"missing fields", `{}`, 0, "", false}, // zero values
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Money
			err := m.UnmarshalJSON([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAmount, m.Amount())
				assert.Equal(t, tt.wantCurr, m.Currency())
			}
		})
	}
}
