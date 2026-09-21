package bogus

import (
	"context"
	"testing"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/gateway"
)

// import_gateway checks that the "bogus" factory is registered and operational.
func import_gateway(t *testing.T) {
	t.Helper()
	gw, err := gateway.New("bogus", nil)
	if err != nil {
		t.Fatalf("gateway.New(%q): %v", "bogus", err)
	}
	if gw.Name() != "bogus" {
		t.Errorf("gateway Name() = %q; want %q", gw.Name(), "bogus")
	}
	// Quick smoke test via the registered factory.
	resp, err := gw.Purchase(context.Background(), &paykit.PurchaseRequest{Amount: 100})
	if err != nil {
		t.Fatalf("Purchase via registry: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success via registered gateway, got failure")
	}
}


// newClient returns a zero-value Client ready for use in tests.
func newClient(t *testing.T) *Client {
	t.Helper()
	return &Client{}
}

// bg returns a background context — a shorthand used across test cases.
func bg() context.Context { return context.Background() }

// purchaseReq builds a PurchaseRequest with the given amount.
func purchaseReq(amount int) *paykit.PurchaseRequest {
	return &paykit.PurchaseRequest{
		Amount:         amount,
		Currency:       "KES",
		Phone:          "254712345678",
		Description:    "test purchase",
		IdempotencyKey: "idem-purchase",
	}
}

// authorizeReq builds an AuthorizeRequest with the given amount.
func authorizeReq(amount int) *paykit.AuthorizeRequest {
	return &paykit.AuthorizeRequest{
		Amount:         amount,
		Currency:       "KES",
		Phone:          "254712345678",
		Description:    "test authorize",
		IdempotencyKey: "idem-authorize",
	}
}

// captureReq builds a CaptureRequest with the given amount.
func captureReq(txID string, amount int) *paykit.CaptureRequest {
	return &paykit.CaptureRequest{
		TransactionID:  txID,
		Amount:         amount,
		IdempotencyKey: "idem-capture",
	}
}

// voidReq builds a VoidRequest for the given transaction ID.
func voidReq(txID string) *paykit.VoidRequest {
	return &paykit.VoidRequest{
		TransactionID:  txID,
		IdempotencyKey: "idem-void",
	}
}

// refundReq builds a RefundRequest with the given amount.
func refundReq(txID string, amount int) *paykit.RefundRequest {
	return &paykit.RefundRequest{
		TransactionID:  txID,
		Amount:         amount,
		IdempotencyKey: "idem-refund",
	}
}

// statusReq builds a StatusRequest for the given transaction ID.
func statusReq(txID string) *paykit.StatusRequest {
	return &paykit.StatusRequest{
		TransactionID:  txID,
		IdempotencyKey: "idem-status",
	}
}
