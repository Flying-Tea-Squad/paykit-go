package pesapal

import (
	"encoding/json"
	"testing"
)

func TestAuthRequestJSON(t *testing.T) {
	request := AuthRequest{
		ConsumerKey:    "test-key",
		ConsumerSecret: "test-secret",
	}

	got, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	want := `{"consumer_key":"test-key","consumer_secret":"test-secret"}`
	if string(got) != want {
		t.Fatalf("json.Marshal() = %s, want %s", got, want)
	}
}

func TestAuthResponseJSON(t *testing.T) {
	payload := `{
		"token": "test-token",
		"expiryDate": "2026-10-29T12:00:00Z",
		"status": "200",
		"message": "Request processed successfully"
	}`

	var response AuthResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.Token != "test-token" {
		t.Errorf("Token = %q, want %q", response.Token, "test-token")
	}

	if response.Error != nil {
		t.Errorf("Error = %v, want nil", response.Error)
	}
}

// TestAuthRequestRequiredFields verifies that authentication credentials
// are included in the JSON request.
func TestAuthRequestRequiredFields(t *testing.T) {
	request := AuthRequest{
		ConsumerKey:    "test-key",
		ConsumerSecret: "test-secret",
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	required := []string{"consumer_key", "consumer_secret"}
	for _, field := range required {
		value, exists := payload[field]
		if !exists {
			t.Errorf("required field %q is missing from JSON", field)
			continue
		}

		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			t.Errorf("field %q should be a string: %v", field, err)
			continue
		}

		if text == "" {
			t.Errorf("required field %q is empty", field)
		}
	}
}

// TestSubmitOrderRequestRequiredFields verifies that required order
// fields are present and non-empty in the serialized request.
func TestSubmitOrderRequestRequiredFields(t *testing.T) {
	request := SubmitOrderRequest{
		ID:             "order-123",
		Currency:       "KES",
		Amount:         1500.00,
		Description:    "Test payment",
		CallbackURL:    "https://example.com/callback",
		NotificationID: "ipn-test-123",
		BillingAddress: BillingAddress{
			EmailAddress: "customer@example.com",
			FirstName:    "Test",
			LastName:     "Customer",
			CountryCode:  "KE",
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	required := []string{
		"id",
		"currency",
		"amount",
		"description",
		"callback_url",
		"notification_id",
		"billing_address",
	}

	for _, field := range required {
		value, exists := payload[field]
		if !exists {
			t.Errorf("required field %q is missing from JSON", field)
			continue
		}

		if string(value) == `""` || string(value) == "null" {
			t.Errorf("required field %q has an empty value", field)
		}
	}

	var address map[string]json.RawMessage
	if err := json.Unmarshal(payload["billing_address"], &address); err != nil {
		t.Fatalf("billing_address should be a JSON object: %v", err)
	}

	if len(address) == 0 {
		t.Error("billing_address should not be empty")
	}
}
