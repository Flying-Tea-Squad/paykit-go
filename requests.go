package paykit

type PurchaseRequest struct {
	IdempotencyKey string
}

type AuthorizeRequest struct {
	IdempotencyKey string
}

type CaptureRequest struct {
	IdempotencyKey string
}

type VoidRequest struct {
	IdempotencyKey string
}

type RefundRequest struct {
	IdempotencyKey string
}

type StatusRequest struct {
	IdempotencyKey string
}
