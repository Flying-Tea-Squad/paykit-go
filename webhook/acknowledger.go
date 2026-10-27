package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

// Acknowledger defines the capability to construct and write provider-specific HTTP acknowledgment
// responses for successfully verified and handled payment webhooks.
//
// Telco payment rails require explicit acknowledgment formats to avoid repeated retry loops.
type Acknowledger interface {
	// Acknowledge writes provider-specific response headers, status codes, and payloads to w.
	// Any caller-facing side effect (such as flushing headers or emitting bytes) occurs immediately.
	Acknowledge(w http.ResponseWriter, r *http.Request, event *paykit.Event) error
}

// AcknowledgeFunc is an adapter allowing the use of ordinary functions as webhook acknowledgers.
type AcknowledgeFunc func(w http.ResponseWriter, r *http.Request, event *paykit.Event) error

// Acknowledge calls f(w, r, event).
func (f AcknowledgeFunc) Acknowledge(w http.ResponseWriter, r *http.Request, event *paykit.Event) error {
	return f(w, r, event)
}

// MpesaAcknowledgment represents the standard Safaricom Daraja acknowledgment envelope.
// Safaricom requires a 200 OK response containing this JSON structure to mark STK Push,
// C2B, and B2C notifications as delivered.
type MpesaAcknowledgment struct {
	ResultCode int    `json:"ResultCode"`
	ResultDesc string `json:"ResultDesc"`
}

// AirtelAcknowledgment represents the standard Airtel Money callback acknowledgment envelope.
type AirtelAcknowledgment struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// PesapalAcknowledgment represents the standard Pesapal IPN callback acknowledgment envelope.
type PesapalAcknowledgment struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// DefaultAcknowledgment represents the generic JSON callback acknowledgment envelope for
// payment providers without proprietary response specifications.
type DefaultAcknowledgment struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// writeJSON writes the HTTP status, Content-Type header, and JSON-encoded payload to w.
func writeJSON(w http.ResponseWriter, statusCode int, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return fmt.Errorf("webhook: failed to write acknowledgment JSON: %w", err)
	}
	return nil
}

// defaultMpesaAcknowledger writes the Safaricom Daraja JSON acknowledgment.
var defaultMpesaAcknowledger = AcknowledgeFunc(func(w http.ResponseWriter, _ *http.Request, _ *paykit.Event) error {
	return writeJSON(w, http.StatusOK, MpesaAcknowledgment{
		ResultCode: 0,
		ResultDesc: "Accepted",
	})
})

// defaultAirtelAcknowledger writes the Airtel Money JSON acknowledgment.
var defaultAirtelAcknowledger = AcknowledgeFunc(func(w http.ResponseWriter, _ *http.Request, _ *paykit.Event) error {
	return writeJSON(w, http.StatusOK, AirtelAcknowledgment{
		Status:  "success",
		Message: "Accepted",
	})
})

// defaultPesapalAcknowledger writes the Pesapal IPN JSON acknowledgment.
var defaultPesapalAcknowledger = AcknowledgeFunc(func(w http.ResponseWriter, _ *http.Request, _ *paykit.Event) error {
	return writeJSON(w, http.StatusOK, PesapalAcknowledgment{
		Status:  "200",
		Message: "OK",
	})
})

// defaultGeneralAcknowledger writes the generic JSON acknowledgment.
var defaultGeneralAcknowledger = AcknowledgeFunc(func(w http.ResponseWriter, _ *http.Request, _ *paykit.Event) error {
	return writeJSON(w, http.StatusOK, DefaultAcknowledgment{
		Status:  "success",
		Message: "Webhook processed successfully",
	})
})
