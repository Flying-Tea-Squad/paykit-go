// Package bogus provides a deterministic test-double payment gateway.
//
// Behaviour rules (keyed on the Amount field, or TransactionID for QueryStatus):
//
//   - Amount % 100 == 0  → success
//   - Amount % 100 == 5  → failure with ErrCardDeclined
//
// The gateway registers itself under the name "bogus" at program startup via init().
// Import it with a blank identifier to activate the registration:
//
//	import _ "github.com/Flying-Tea-Squad/paykit-go/bogus"
package bogus

import (
	"context"
	"fmt"
	"strings"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/gateway"
)

func init() {
	gateway.Register("bogus", func(config any) (paykit.Gateway, error) {
		return &Client{}, nil
	})
}

// Client is the bogus gateway.  Its zero value is ready for use.
type Client struct{}

// Name returns the provider identifier.
func (c *Client) Name() string { return "bogus" }

// Purchase authorises and captures in a single step.
func (c *Client) Purchase(ctx context.Context, req *paykit.PurchaseRequest) (*paykit.Response, error) {
	return outcome(req.Amount)
}

// Authorize places a hold on funds without capturing.
func (c *Client) Authorize(ctx context.Context, req *paykit.AuthorizeRequest) (*paykit.Response, error) {
	return outcome(req.Amount)
}

// Capture completes a previously authorised transaction.
func (c *Client) Capture(ctx context.Context, req *paykit.CaptureRequest) (*paykit.Response, error) {
	return outcome(req.Amount)
}

// Void cancels a previously authorised transaction.
// Amount is taken from the Metadata field "amount" if present; otherwise defaults to 0 (success).
func (c *Client) Void(ctx context.Context, req *paykit.VoidRequest) (*paykit.Response, error) {
	// Void does not naturally carry an Amount; derive behaviour from TransactionID.
	return outcomeFromTxID(req.TransactionID)
}

// Refund reverses a completed transaction.
func (c *Client) Refund(ctx context.Context, req *paykit.RefundRequest) (*paykit.Response, error) {
	return outcome(req.Amount)
}

// QueryStatus returns the current status of a transaction.
// A TransactionID prefixed with "fail" is treated as a declined transaction;
// any other non-empty ID is treated as successful.
func (c *Client) QueryStatus(ctx context.Context, req *paykit.StatusRequest) (*paykit.Response, error) {
	return outcomeFromTxID(req.TransactionID)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// outcome decides success/failure based on the last two digits of amount.
func outcome(amount int) (*paykit.Response, error) {
	switch amount % 100 {
	case 0:
		return &paykit.Response{
			Success:       true,
			Message:       "approved",
			TransactionID: fmt.Sprintf("bogus_%d", amount),
		}, nil
	case 5:
		return &paykit.Response{
			Success:   false,
			Message:   "card declined",
			ErrorCode: paykit.ErrCardDeclined,
		}, paykit.ErrCardDeclinedSentinel
	default:
		return &paykit.Response{
			Success:   false,
			Message:   "unrecognised amount pattern",
			ErrorCode: paykit.ErrProcessingError,
		}, paykit.ErrProcessingErrorSentinel
	}
}

// outcomeFromTxID derives success/failure from a transaction ID.
// IDs prefixed with "fail" → declined; everything else → success.
func outcomeFromTxID(txID string) (*paykit.Response, error) {
	if strings.HasPrefix(txID, "fail") {
		return &paykit.Response{
			Success:       false,
			Message:       "card declined",
			ErrorCode:     paykit.ErrCardDeclined,
			TransactionID: txID,
		}, paykit.ErrCardDeclinedSentinel
	}
	return &paykit.Response{
		Success:       true,
		Message:       "approved",
		TransactionID: txID,
	}, nil
}
