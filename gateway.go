package paykit

import (
	"context"
	"net/http"
)

// Gateway is the primary collection interface implemented by providers that
// support customer charges, refunds, and transaction status queries.
type Gateway interface {
	// Name returns the canonical provider identifier (e.g., "mpesa", "airtel", "pesapal").
	Name() string

	// Charge initiates a customer payment collection (e.g., STK Push, USSD push, or hosted checkout).
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)

	// QueryStatus queries the current status of an initiated transaction.
	QueryStatus(ctx context.Context, req *StatusRequest) (*StatusResponse, error)

	// Refund initiates a full or partial refund/reversal of a completed charge.
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
}

// Disburser is implemented by payment providers supporting B2C payouts.
type Disburser interface {
	// Disburse transfers funds from the business account to a recipient (e.g. B2C mobile payout).
	Disburse(ctx context.Context, req *DisbursementRequest) (*DisbursementResponse, error)
}

// WebhookHandler is implemented by providers that receive and process asynchronous callbacks.
type WebhookHandler interface {
	// ParseAndVerify validates request authenticity (signatures, tokens) and unpacks the payload into an Event.
	ParseAndVerify(r *http.Request) (*Event, error)
}

// BalanceChecker is implemented by providers that expose account balance inquiries.
type BalanceChecker interface {
	// CheckBalance retrieves current ledger and available balances from the provider.
	CheckBalance(ctx context.Context, req *BalanceRequest) (*BalanceResponse, error)
}

// Event represents a parsed webhook event.
type Event struct {
	Type      string
	TransactionID string
	Status    TransactionStatus
	Amount    Money
	Raw       map[string]any
}

// BalanceRequest queries account balance.
type BalanceRequest struct {
	IdempotencyKey string
}

// BalanceResponse represents balance inquiry result.
type BalanceResponse struct {
	LedgerBalance   Money
	AvailableBalance Money
	Raw             map[string]any
}