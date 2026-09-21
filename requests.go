package paykit

// PurchaseRequest represents a direct charge / payment request.
type PurchaseRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

// AuthorizeRequest holds-but-does-not-capture funds.
type AuthorizeRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

// CaptureRequest captures a previously authorized amount.
type CaptureRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
	Metadata       map[string]string
}

// VoidRequest cancels a previously authorized transaction.
type VoidRequest struct {
	TransactionID  string
	IdempotencyKey string
	Metadata       map[string]string
}

// RefundRequest reverses a completed transaction.
type RefundRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
	Metadata       map[string]string
}

// StatusRequest queries the status of a transaction.
type StatusRequest struct {
	TransactionID  string
	IdempotencyKey string
	Metadata       map[string]string
}
