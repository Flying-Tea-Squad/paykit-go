package paykit

import "encoding/json"

// Response represents the standardized response returned by all payment providers.
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
