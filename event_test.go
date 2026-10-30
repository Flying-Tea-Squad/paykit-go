package paykit

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTime = time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)

func TestEvent_HelperMethods(t *testing.T) {
	t.Parallel()

	t.Run("nil safety", func(t *testing.T) {
		t.Parallel()

		var nilEvent *Event

		assert.False(t, nilEvent.IsSuccess(), "IsSuccess should be false for nil")
		assert.False(t, nilEvent.IsFailed(), "IsFailed should be false for nil")
		assert.False(t, nilEvent.IsCharge(), "IsCharge should be false for nil")
		assert.False(t, nilEvent.IsDisbursement(), "IsDisbursement should be false for nil")
	})

	tests := []struct {
		name           string
		event          *Event
		expectSuccess  bool
		expectFailed   bool
		expectCharge   bool
		expectDisburse bool
	}{
		{
			name:          "charge completed",
			event:         &Event{Type: EventChargeCompleted, Status: StatusSuccess},
			expectSuccess: true,
			expectCharge:  true,
		},
		{
			name:         "charge failed",
			event:        &Event{Type: EventChargeFailed, Status: StatusFailed},
			expectFailed: true,
			expectCharge: true,
		},
		{
			name:           "disbursement completed",
			event:          &Event{Type: EventDisbursementCompleted, Status: StatusSuccess},
			expectSuccess:  true,
			expectDisburse: true,
		},
		{
			name:           "disbursement failed",
			event:          &Event{Type: EventDisbursementFailed, Status: StatusFailed},
			expectFailed:   true,
			expectDisburse: true,
		},
		{
			name:          "c2b received is neither charge nor disbursement",
			event:         &Event{Type: EventC2BReceived, Status: StatusSuccess},
			expectSuccess: true,
		},
		{
			name:          "unknown event type is neither charge nor disbursement",
			event:         &Event{Type: EventType("refund.processed"), Status: StatusSuccess},
			expectSuccess: true,
		},
		{
			name:          "empty event type is neither charge nor disbursement",
			event:         &Event{Status: StatusSuccess},
			expectSuccess: true,
		},
		{
			name:         "empty status is neither success nor failure",
			event:        &Event{Type: EventChargeCompleted},
			expectCharge: true,
		},
		{
			name:         "unknown status is neither success nor failure",
			event:        &Event{Type: EventChargeCompleted, Status: TransactionStatus("processing")},
			expectCharge: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectSuccess, tt.event.IsSuccess(), "IsSuccess")
			assert.Equal(t, tt.expectFailed, tt.event.IsFailed(), "IsFailed")
			assert.Equal(t, tt.expectCharge, tt.event.IsCharge(), "IsCharge")
			assert.Equal(t, tt.expectDisburse, tt.event.IsDisbursement(), "IsDisbursement")
		})
	}
}

func TestEvent_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		jsonInput   string
		expectErr   bool
		assertErr   func(t *testing.T, err error)
		assertEvent func(t *testing.T, e *Event)
	}{
		{
			name: "full valid payload",
			jsonInput: `{
				"id": "evt_123",
				"provider": "mpesa",
				"type": "charge.completed",
				"transaction_id": "QKH123",
				"reference": "order_1",
				"amount": {"amount": 150000, "currency": "KES"},
				"phone": "254700000000",
				"timestamp": "2026-10-29T10:00:00Z",
				"status": "success",
				"raw": {"telco": "data"},
				"metadata": {"foo": "bar"}
			}`,
			assertEvent: func(t *testing.T, e *Event) {
				assert.Equal(t, "evt_123", e.ID)
				assert.Equal(t, "mpesa", e.Provider)
				assert.Equal(t, EventChargeCompleted, e.Type)
				assert.Equal(t, "QKH123", e.TransactionID)
				assert.Equal(t, "order_1", e.Reference)
				assert.Equal(t, int64(150000), e.Amount.Amount())
				assert.Equal(t, "KES", e.Amount.Currency())
				assert.Equal(t, "254700000000", e.Phone)
				assert.Equal(t, testTime, e.Timestamp)
				assert.Equal(t, StatusSuccess, e.Status)
				assert.Equal(t, "bar", e.Metadata["foo"])
				assertRawJSONEqual(t, e.Raw, `{"telco":"data"}`)
			},
		},
		{
			name: "minimal valid payload omits optional fields",
			jsonInput: `{
				"provider": "airtel",
				"type": "c2b.received",
				"amount": {"amount": 5000, "currency": "KES"},
				"timestamp": "2026-10-29T10:00:00Z"
			}`,
			assertEvent: func(t *testing.T, e *Event) {
				assert.Equal(t, "airtel", e.Provider)
				assert.Equal(t, EventC2BReceived, e.Type)
				assert.Equal(t, int64(5000), e.Amount.Amount())
				assert.Equal(t, "KES", e.Amount.Currency())
				assert.Equal(t, testTime, e.Timestamp)

				assert.Empty(t, e.ID)
				assert.Empty(t, e.TransactionID)
				assert.Empty(t, e.Reference)
				assert.Empty(t, e.Phone)
				assert.Empty(t, e.Status)
				assert.Nil(t, e.Raw)
				assert.Nil(t, e.Metadata)
			},
		},
		{
			name: "null optional fields remain nil or literal null",
			jsonInput: `{
				"provider": "mpesa",
				"type": "charge.completed",
				"amount": {"amount": 100, "currency": "KES"},
				"timestamp": "2026-10-29T10:00:00Z",
				"raw": null,
				"metadata": null
			}`,
			assertEvent: func(t *testing.T, e *Event) {
				assert.Equal(t, json.RawMessage("null"), e.Raw)
				assert.Nil(t, e.Metadata)
			},
		},
		{
			name:      "invalid amount type",
			jsonInput: `{"provider":"mpesa","type":"charge.completed","amount":"not_an_object"}`,
			expectErr: true,
		},
		{
			name:      "missing currency in amount results in empty currency",
			jsonInput: `{"provider":"mpesa","type":"charge.completed","amount":{"amount":100},"timestamp":"2026-10-29T10:00:00Z"}`,
			assertEvent: func(t *testing.T, e *Event) {
				assert.Equal(t, int64(100), e.Amount.Amount())
				assert.Empty(t, e.Amount.Currency(), "Money currently accepts missing currencies as empty string")
			},
		},
		{
			name:      "invalid timestamp",
			jsonInput: `{"provider":"mpesa","type":"charge.completed","timestamp":"not-a-time"}`,
			expectErr: true,
		},
		{
			name:      "malformed raw JSON causes syntax error",
			jsonInput: `{"provider":"mpesa","type":"charge.completed","raw": {invalid}}`,
			expectErr: true,
			assertErr: func(t *testing.T, err error) {
				var syntaxErr *json.SyntaxError
				require.ErrorAs(t, err, &syntaxErr, "expected a json.SyntaxError for malformed raw JSON")
			},
		},
		{
			name: "unknown top-level field is tolerated",
			jsonInput: `{
				"provider": "mpesa",
				"type": "charge.completed",
				"amount": {"amount": 100, "currency": "KES"},
				"timestamp": "2026-10-29T10:00:00Z",
				"provider_extension": {"foo": "bar"}
			}`,
			assertEvent: func(t *testing.T, e *Event) {
				assert.Equal(t, "mpesa", e.Provider)
				assert.Equal(t, EventChargeCompleted, e.Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var event Event
			err := json.Unmarshal([]byte(tt.jsonInput), &event)

			if tt.expectErr {
				require.Error(t, err)
				if tt.assertErr != nil {
					tt.assertErr(t, err)
				}
				return
			}

			require.NoError(t, err)

			if tt.assertEvent != nil {
				tt.assertEvent(t, &event)
			}
		})
	}
}

func TestEvent_JSONMarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		event       *Event
		wantPresent []string
		wantAbsent  []string
		assertJSON  func(t *testing.T, decoded map[string]any)
	}{
		{
			name: "minimal event omits optional fields",
			event: &Event{
				Provider:  "pesapal",
				Type:      EventDisbursementFailed,
				Amount:    NewMoney(5000, "KES"),
				Timestamp: testTime,
			},
			wantPresent: []string{"provider", "type", "amount", "timestamp"},
			wantAbsent: []string{
				"id",
				"transaction_id",
				"reference",
				"phone",
				"status",
				"raw",
				"metadata",
			},
		},
		{
			name: "populated optional fields are emitted",
			event: &Event{
				ID:            "evt_123",
				Provider:      "mpesa",
				Type:          EventChargeCompleted,
				TransactionID: "QKH123",
				Reference:     "order_1",
				Amount:        NewMoney(150000, "KES"),
				Phone:         "254700000000",
				Timestamp:     testTime,
				Status:        StatusSuccess,
				Raw:           json.RawMessage(`{"signature":"abc123"}`),
				Metadata:      map[string]any{"foo": "bar"},
			},
			wantPresent: []string{
				"id",
				"provider",
				"type",
				"transaction_id",
				"reference",
				"amount",
				"phone",
				"timestamp",
				"status",
				"raw",
				"metadata",
			},
			assertJSON: func(t *testing.T, decoded map[string]any) {
				raw, ok := decoded["raw"].(map[string]any)
				require.True(t, ok, "raw should be a JSON object, not a base64 string")
				assert.Equal(t, "abc123", raw["signature"])

				metadata, ok := decoded["metadata"].(map[string]any)
				require.True(t, ok, "metadata should be a JSON object")
				assert.Equal(t, "bar", metadata["foo"])

				assert.Equal(t, "2026-10-29T10:00:00Z", decoded["timestamp"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.event)
			require.NoError(t, err)

			var decoded map[string]any
			require.NoError(t, json.Unmarshal(data, &decoded))

			for _, field := range tt.wantPresent {
				assert.Contains(t, decoded, field)
			}

			for _, field := range tt.wantAbsent {
				assert.NotContains(t, decoded, field)
			}

			if tt.assertJSON != nil {
				tt.assertJSON(t, decoded)
			}
		})
	}
}

func TestEvent_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := &Event{
		ID:            "evt_123",
		Provider:      "mpesa",
		Type:          EventChargeCompleted,
		TransactionID: "QKH123",
		Reference:     "order_1",
		Amount:        NewMoney(150000, "KES"),
		Phone:         "254700000000",
		Timestamp:     testTime,
		Status:        StatusSuccess,
		Raw:           json.RawMessage(`{"telco":"data"}`),
		Metadata:      map[string]any{"foo": "bar"},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Event
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Provider, decoded.Provider)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.TransactionID, decoded.TransactionID)
	assert.Equal(t, original.Reference, decoded.Reference)
	assert.Equal(t, original.Amount.Amount(), decoded.Amount.Amount())
	assert.Equal(t, original.Amount.Currency(), decoded.Amount.Currency())
	assert.Equal(t, original.Phone, decoded.Phone)
	assert.Equal(t, original.Timestamp, decoded.Timestamp)
	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.Metadata, decoded.Metadata)
	assertRawJSONEqual(t, decoded.Raw, string(original.Raw))
}

// TestEvent_RawPreservesProviderBytes proves that json.RawMessage preserves
// the exact key order and structure across a marshal/unmarshal round-trip.
// Note: Go's encoding/json may strip whitespace during unmarshaling into RawMessage,
// but structural fidelity (key order) is strictly maintained, which is the
// critical requirement for audit replays. (Signature verification should always
// be performed on the raw HTTP request body before unmarshaling).
func TestEvent_RawPreservesProviderBytes(t *testing.T) {
	t.Parallel()

	originalBytes := `{"telco": "data", "b": 1, "a": 2}`
	original := json.RawMessage(originalBytes)

	event := &Event{
		Provider:  "mpesa",
		Type:      EventChargeCompleted,
		Amount:    NewMoney(100, "KES"),
		Timestamp: testTime,
		Raw:       original,
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded Event
	require.NoError(t, json.Unmarshal(data, &decoded))

	// Go's encoding/json preserves key order but may strip whitespace.
	// We compact the original to compare structural fidelity.
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, original))

	assert.Equal(t, buf.String(), string(decoded.Raw))
}

func assertRawJSONEqual(t *testing.T, raw json.RawMessage, expected string) {
	t.Helper()

	var got, want any

	require.NoError(t, json.Unmarshal(raw, &got), "actual raw value should be valid JSON")
	require.NoError(t, json.Unmarshal([]byte(expected), &want), "expected raw value should be valid JSON")

	assert.Equal(t, want, got)
}
