package paykit

// ChargeRequest initiates a customer payment collection (e.g., STK Push, USSD push, hosted checkout).
type ChargeRequest struct {
	// Amount is the transaction amount in minor units (cents).
	Amount Money
	// Phone is the customer's phone number in E.164 format (e.g., 2547XXXXXXXX).
	Phone string
	// Description is a human-readable description of the charge.
	Description string
	// IdempotencyKey is a unique key to prevent duplicate charges.
	IdempotencyKey string
	// CallbackURL is the URL to receive asynchronous payment notifications.
	CallbackURL string
	// Metadata stores additional provider-specific information.
	Metadata map[string]string
}

// DisbursementRequest initiates a business-to-customer (B2C) payout.
type DisbursementRequest struct {
	// Amount is the payout amount in minor units (cents).
	Amount Money
	// Phone is the recipient's phone number in E.164 format (e.g., 2547XXXXXXXX).
	Phone string
	// Description is a human-readable description of the disbursement.
	Description string
	// IdempotencyKey is a unique key to prevent duplicate disbursements.
	IdempotencyKey string
	// CallbackURL is the URL to receive asynchronous payout notifications.
	CallbackURL string
	// Metadata stores additional provider-specific information.
	Metadata map[string]string
}

// StatusRequest queries the current state of a transaction.
type StatusRequest struct {
	// TransactionID is the provider's transaction identifier.
	TransactionID string
	// IdempotencyKey is an optional key for idempotent status queries.
	IdempotencyKey string
}

// RefundRequest initiates a full or partial refund/reversal of a completed charge.
type RefundRequest struct {
	// TransactionID is the provider's transaction identifier of the original charge.
	TransactionID string
	// Amount is the refund amount in minor units. Zero or omitted means full refund.
	Amount Money
	// IdempotencyKey is a unique key to prevent duplicate refunds.
	IdempotencyKey string
}
