package paykit_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Flying-Tea-Squad/paykit-go"
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
			m := paykit.NewMoney(tt.minorUnits, tt.currency)
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
			m := paykit.NewKES(tt.shillings)
			assert.Equal(t, tt.wantAmount, m.Amount())
			assert.Equal(t, "KES", m.Currency())
		})
	}
}

func TestMoney_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		m        paykit.Money
		expected bool
	}{
		{"zero amount", paykit.NewMoney(0, "KES"), true},
		{"positive amount", paykit.NewMoney(1, "KES"), false},
		{"negative amount", paykit.NewMoney(-1, "KES"), false},
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
		m        paykit.Money
		expected string
	}{
		{"KES whole", paykit.NewMoney(10000, "KES"), "KES 100.00"},
		{"KES with cents", paykit.NewMoney(10050, "KES"), "KES 100.50"},
		{"USD", paykit.NewMoney(5000, "USD"), "USD 50.00"},
		{"zero", paykit.NewMoney(0, "KES"), "KES 0.00"},
		{"negative", paykit.NewMoney(-2500, "EUR"), "EUR -25.00"},
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
		m        paykit.Money
		expected float64
	}{
		{"whole", paykit.NewMoney(10000, "KES"), 100.0},
		{"with cents", paykit.NewMoney(10050, "KES"), 100.5},
		{"zero", paykit.NewMoney(0, "KES"), 0.0},
		{"negative", paykit.NewMoney(-5000, "USD"), -50.0},
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
		a          paykit.Money
		b          paykit.Money
		wantAmount int64
		wantPanic  bool
	}{
		{"same currency", paykit.NewMoney(10000, "KES"), paykit.NewMoney(5000, "KES"), 15000, false},
		{"zero addend", paykit.NewMoney(10000, "KES"), paykit.NewMoney(0, "KES"), 10000, false},
		{"negative addend", paykit.NewMoney(10000, "KES"), paykit.NewMoney(-3000, "KES"), 7000, false},
		{"different currency panics", paykit.NewMoney(10000, "KES"), paykit.NewMoney(5000, "USD"), 0, true},
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
		a          paykit.Money
		b          paykit.Money
		wantAmount int64
		wantPanic  bool
	}{
		{"same currency", paykit.NewMoney(10000, "KES"), paykit.NewMoney(3000, "KES"), 7000, false},
		{"zero subtrahend", paykit.NewMoney(10000, "KES"), paykit.NewMoney(0, "KES"), 10000, false},
		{"negative result", paykit.NewMoney(3000, "KES"), paykit.NewMoney(10000, "KES"), -7000, false},
		{"different currency panics", paykit.NewMoney(10000, "KES"), paykit.NewMoney(5000, "USD"), 0, true},
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
		m          paykit.Money
		factor     int64
		wantAmount int64
	}{
		{"multiply by 2", paykit.NewMoney(5000, "KES"), 2, 10000},
		{"multiply by 0", paykit.NewMoney(5000, "KES"), 0, 0},
		{"multiply by negative", paykit.NewMoney(5000, "KES"), -3, -15000},
		{"zero amount", paykit.NewMoney(0, "KES"), 10, 0},
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
		a         paykit.Money
		b         paykit.Money
		want      int
		wantPanic bool
	}{
		{"less than", paykit.NewMoney(5000, "KES"), paykit.NewMoney(10000, "KES"), -1, false},
		{"equal", paykit.NewMoney(5000, "KES"), paykit.NewMoney(5000, "KES"), 0, false},
		{"greater than", paykit.NewMoney(10000, "KES"), paykit.NewMoney(5000, "KES"), 1, false},
		{"negative vs positive", paykit.NewMoney(-5000, "KES"), paykit.NewMoney(5000, "KES"), -1, false},
		{"different currency panics", paykit.NewMoney(5000, "KES"), paykit.NewMoney(5000, "USD"), 0, true},
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
		status   paykit.TransactionStatus
		expected bool
	}{
		{"pending_action not terminal", paykit.StatusPendingAction, false},
		{"processing not terminal", paykit.StatusProcessing, false},
		{"success terminal", paykit.StatusSuccess, true},
		{"failed terminal", paykit.StatusFailed, true},
		{"cancelled terminal", paykit.StatusCancelled, true},
		{"timeout terminal", paykit.StatusTimeout, true},
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
		status   paykit.TransactionStatus
		expected string
	}{
		{"pending_action", paykit.StatusPendingAction, "pending_action"},
		{"processing", paykit.StatusProcessing, "processing"},
		{"success", paykit.StatusSuccess, "success"},
		{"failed", paykit.StatusFailed, "failed"},
		{"cancelled", paykit.StatusCancelled, "cancelled"},
		{"timeout", paykit.StatusTimeout, "timeout"},
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
		expected paykit.TransactionStatus
	}{
		{"pending_action exact", "pending_action", paykit.StatusPendingAction},
		{"pending alias", "pending", paykit.StatusPendingAction},
		{"processing exact", "processing", paykit.StatusProcessing},
		{"in_progress alias", "in_progress", paykit.StatusProcessing},
		{"success exact", "success", paykit.StatusSuccess},
		{"completed alias", "completed", paykit.StatusSuccess},
		{"complete alias", "complete", paykit.StatusSuccess},
		{"failed exact", "failed", paykit.StatusFailed},
		{"failure alias", "failure", paykit.StatusFailed},
		{"declined alias", "declined", paykit.StatusFailed},
		{"cancelled exact", "cancelled", paykit.StatusCancelled},
		{"canceled US spelling", "canceled", paykit.StatusCancelled},
		{"voided alias", "voided", paykit.StatusCancelled},
		{"timeout exact", "timeout", paykit.StatusTimeout},
		{"timed_out alias", "timed_out", paykit.StatusTimeout},
		{"case insensitive", "SUCCESS", paykit.StatusSuccess},
		{"with whitespace", "  pending  ", paykit.StatusPendingAction},
		{"unknown defaults to unknown", "unknown_status", paykit.StatusUnknown},
		{"empty defaults to unknown", "", paykit.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, paykit.ParseTransactionStatus(tt.input))
		})
	}
}

func TestMoney_MarshalJSON(t *testing.T) {
	m := paykit.NewMoney(10050, "KES")
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
			var m paykit.Money
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
