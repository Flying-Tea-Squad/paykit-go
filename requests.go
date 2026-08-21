package paykit

type PurchaseRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

type AuthorizeRequest struct {
	Amount         int
	Currency       string
	Phone          string
	Description    string
	IdempotencyKey string
	CallbackURL    string
	Metadata       map[string]string
}

type CaptureRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
}

type VoidRequest struct {
	TransactionID  string
	IdempotencyKey string
}

type RefundRequest struct {
	TransactionID  string
	Amount         int
	IdempotencyKey string
}

type StatusRequest struct {
	TransactionID  string
	IdempotencyKey string
}
