package paykit

import "context"

// Gateway defines the common operations that every payment provider
// must implement.
type Gateway interface {
	Purchase(ctx context.Context, req *PurchaseRequest) (*Response, error)
	Authorize(ctx context.Context, req *AuthorizeRequest) (*Response, error)
	Capture(ctx context.Context, req *CaptureRequest) (*Response, error)
	Void(ctx context.Context, req *VoidRequest) (*Response, error)
	Refund(ctx context.Context, req *RefundRequest) (*Response, error)
	QueryStatus(ctx context.Context, req *StatusRequest) (*Response, error)

	// Name returns the provider name (e.g. mpesa, airtelmoney, pesapal).
	Name() string
}
