package paykit

// ChargeRequest encapsulates parameters required to initiate a customer payment collection
// (e.g. SIM Toolkit STK Push, USSD prompt, or hosted order checkout).
type ChargeRequest struct {
	// Amount is the payment sum in the smallest fractional currency unit (e.g., Kenyan cents).
	Amount int `json:"amount"`

	// Currency is the standard three-letter ISO 4217 currency code (e.g. "KES", "UGX", "USD").
	Currency string `json:"currency"`

	// Phone is the customer MSISDN formatted in E.164 (e.g. "+254712345678").
	Phone string `json:"phone,omitempty"`

	// Description is a human-readable memo or statement descriptor for the charge.
	Description string `json:"description,omitempty"`

	// Reference is the merchant's internal order or account reference identifier.
	Reference string `json:"reference,omitempty"`

	// IdempotencyKey prevents duplicate processing of identical charge requests.
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// CallbackURL is the webhook endpoint where the provider posts asynchronous payment results.
	CallbackURL string `json:"callback_url,omitempty"`

	// Metadata contains optional provider-specific or custom merchant key-value parameters.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// DisbursementRequest encapsulates parameters required to disburse funds to a recipient
// (e.g. B2C mobile money payout, salary distribution, or vendor payment).
type DisbursementRequest struct {
	// Amount is the disbursement sum in minor currency units.
	Amount int `json:"amount"`

	// Currency is the standard three-letter ISO 4217 currency code (e.g. "KES").
	Currency string `json:"currency"`

	// Phone is the recipient MSISDN formatted in E.164.
	Phone string `json:"phone,omitempty"`

	// RecipientName is the optional registered full name of the receiving party.
	RecipientName string `json:"recipient_name,omitempty"`

	// Description is a brief note describing the reason for disbursement.
	Description string `json:"description,omitempty"`

	// Reference is the merchant's internal tracking identifier.
	Reference string `json:"reference,omitempty"`

	// IdempotencyKey prevents duplicate disbursement execution.
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// CallbackURL receives provider delivery notifications for the payout.
	CallbackURL string `json:"callback_url,omitempty"`

	// Metadata holds arbitrary merchant metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// StatusRequest provides parameters needed to query the current state of a transaction.
type StatusRequest struct {
	// TransactionID is the provider's unique transaction reference.
	TransactionID string `json:"transaction_id,omitempty"`

	// Reference is the merchant-supplied reference passed during charge or disbursement.
	Reference string `json:"reference,omitempty"`

	// IdempotencyKey is an optional deduplication key.
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// Metadata holds extra query parameters.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RefundRequest encapsulates data needed to reverse a settled transaction.
type RefundRequest struct {
	// TransactionID is the provider's original transaction identifier to refund.
	TransactionID string `json:"transaction_id"`

	// Amount is the amount to refund in minor units (omit or 0 for full refund where supported).
	Amount int `json:"amount,omitempty"`

	// Currency is the standard three-letter ISO 4217 currency code.
	Currency string `json:"currency,omitempty"`

	// Reason details why the refund is being initiated.
	Reason string `json:"reason,omitempty"`

	// IdempotencyKey prevents duplicate refund requests.
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// Metadata holds optional provider parameters.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BalanceRequest encapsulates parameters for querying account balances.
type BalanceRequest struct {
	// Currency optionally limits the balance check to a specific currency ledger.
	Currency string `json:"currency,omitempty"`

	// Metadata holds provider-specific parameters.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Deprecated: PurchaseRequest is deprecated. Use ChargeRequest instead.
type PurchaseRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

// Deprecated: AuthorizeRequest is deprecated. African mobile money rails do not support two-phase authorizations.
type AuthorizeRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

// Deprecated: CaptureRequest is deprecated. African mobile money rails do not support two-phase authorizations.
type CaptureRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
}

// Deprecated: VoidRequest is deprecated. African mobile money rails do not support authorization voids.
type VoidRequest struct {
	TransactionID  string
	IdempotencyKey string
}
