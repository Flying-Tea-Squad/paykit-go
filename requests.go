package paykit

// PurchaseRequest encapsulates data for a standard purchase transaction.
type PurchaseRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

// AuthorizeRequest encapsulates data for placing a hold on funds without capturing.
type AuthorizeRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

// CaptureRequest provides data needed to capture a previously authorized transaction.
type CaptureRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
}

// VoidRequest provides data needed to cancel a prior authorization.
type VoidRequest struct {
	TransactionID  string
	IdempotencyKey string
}

// RefundRequest encapsulates data to reverse a completed capture or purchase.
type RefundRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
}

// StatusRequest provides data needed to query the current state of a transaction.
type StatusRequest struct {
	TransactionID  string
	IdempotencyKey string
}
