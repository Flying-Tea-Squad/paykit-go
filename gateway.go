package paykit

import (
	"context"
	"net/http"
)

// Gateway is the primary payment collection interface that every provider adapter
// implements. It provides methods for customer payment collection (Charge),
// status queries, and refunds.
type Gateway interface {
	// Name returns the unique canonical provider identifier (e.g., "mpesa", "airtel", "pesapal", "bogus").
	Name() string

	// Charge initiates a customer payment collection (e.g. STK Push, USSD push, or hosted checkout).
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)

	// QueryStatus retrieves the current processing state of an initiated transaction.
	QueryStatus(ctx context.Context, req *StatusRequest) (*StatusResponse, error)

	// Refund initiates a full or partial reversal of a completed charge.
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
}

// Disburser defines the capability to disburse funds to a recipient (e.g. B2C mobile payouts).
// Providers that support disbursements implement this interface. Callers can discover
// this capability at runtime using Go type assertions:
//
//	if d, ok := gw.(paykit.Disburser); ok {
//	    resp, err := d.Disburse(ctx, disburseReq)
//	}
type Disburser interface {
	// Disburse transfers funds from the business account to a recipient.
	Disburse(ctx context.Context, req *DisbursementRequest) (*DisbursementResponse, error)
}

// WebhookHandler defines the capability to parse, authenticate, and unpack incoming
// asynchronous notifications (IPNs or webhooks) from a payment provider into a normalized Event.
//
//	if wh, ok := gw.(paykit.WebhookHandler); ok {
//	    event, err := wh.ParseAndVerify(req)
//	}
type WebhookHandler interface {
	// ParseAndVerify validates request authenticity (e.g., HMAC/RSA signatures, IP whitelists)
	// and deserializes the payload into a normalized Event.
	ParseAndVerify(r *http.Request) (*Event, error)
}

// BalanceChecker defines the capability to query the current balance of the merchant's account.
type BalanceChecker interface {
	// CheckBalance retrieves current ledger and available balance information from the provider.
	CheckBalance(ctx context.Context, req *BalanceRequest) (*BalanceResponse, error)
}
