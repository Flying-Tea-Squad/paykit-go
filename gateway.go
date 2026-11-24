package paykit

import (
	"context"
	"net/http"
)

// Gateway is the primary payment collection interface that every provider adapter
// implements. It provides methods for customer payment collection (Charge),
// transaction status queries (QueryStatus), and reversals (Refund).
//
// # Synchronous Initiation vs. Asynchronous Settlement
//
// On African mobile money rails (e.g., M-Pesa STK Push, Airtel Money USSD Push),
// [Gateway.Charge] initiates a push prompt to the customer's mobile device:
//
//   - An initial successful [Gateway.Charge] call returns a [ChargeResponse] with
//     [TransactionStatus] set to [StatusPendingAction] (awaiting user PIN entry) or
//     [StatusProcessing] (in flight on the provider network). This confirms the request
//     was accepted for processing by the telco gateway, NOT that funds were settled.
//   - Final settlement is communicated asynchronously via provider callbacks (webhooks),
//     delivering a terminal state: [StatusSuccess], [StatusFailed], [StatusCancelled],
//     or [StatusTimeout]. Callers can also poll transaction status using [Gateway.QueryStatus].
//   - Use [TransactionStatus.IsTerminal] to verify whether a transaction has reached
//     its final state.
//
// # Error Semantics: Transport Errors vs. Provider Rejections
//
// Methods on [Gateway] differentiate strictly between transport failures and provider rejections:
//
//   - Go error (err != nil): Returned when the operation could not be completed due to
//     infrastructure or transport issues, such as network timeouts, DNS lookup failures,
//     context cancellations, HTTP 5xx server errors, or local validation failures.
//   - Provider Rejection (err == nil && !resp.Success): Returned when the provider
//     successfully received and processed the request, but rejected the transaction
//     (e.g., customer cancelled prompt, insufficient balance, duplicate transaction).
//     In this case, the response object contains details in [ChargeResponse.Message],
//     normalized [TransactionStatus], and provider-specific error details in
//     [ChargeResponse.Metadata] or [ChargeResponse.Raw]. Use [MapGatewayErrorCode] to map
//     provider error codes to standard PayKit error constants.
//
// # Concurrency Safety
//
// All [Gateway] implementations must be safe for concurrent use across multiple goroutines.
type Gateway interface {
	// Name returns the canonical, lowercase provider identifier (e.g., "mpesa", "airtel", "pesapal", "bogus").
	Name() string

	// Charge initiates a customer payment collection (e.g., STK Push, USSD push prompt, or hosted checkout).
	//
	// On asynchronous rails such as mobile money STK push, a successful call initiates the prompt
	// and returns a non-terminal status ([StatusPendingAction] or [StatusProcessing]). Final settlement
	// is delivered via webhook or queried using [Gateway.QueryStatus].
	//
	// Returns a non-nil error only on transport or infrastructure failures. If the provider rejects
	// the charge, Charge returns a nil error and a [ChargeResponse] with Success set to false.
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)

	// QueryStatus retrieves the current processing state of an initiated transaction.
	//
	// It can be used to reconcile in-flight transactions or poll for final settlement when
	// webhooks are delayed or unavailable.
	QueryStatus(ctx context.Context, req *StatusRequest) (*StatusResponse, error)

	// Refund initiates a full or partial reversal of a settled transaction.
	//
	// Support for refunds varies by provider and transaction type.
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
}

// Disburser defines the capability to disburse funds from the business account
// to a recipient (e.g., B2C mobile payouts, vendor payments, or salary distributions).
//
// Providers that support outbound payouts implement this interface. Callers discover
// this capability at runtime using Go type assertions:
//
//	if d, ok := gw.(paykit.Disburser); ok {
//	    resp, err := d.Disburse(ctx, disburseReq)
//	    if err != nil {
//	        log.Printf("transport error: %v", err)
//	    } else if !resp.Success {
//	        log.Printf("disbursement rejected: %s", resp.Message)
//	    }
//	}
//
// # Synchronous vs. Asynchronous Settlement
//
// Depending on the payment provider, [Disburse] may complete synchronously or return
// an in-flight status ([StatusProcessing]), with final payout confirmation delivered
// via an asynchronous webhook callback.
//
// # Error Semantics
//
// A non-nil error indicates transport, networking, or validation failure. A nil error
// with resp.Success == false indicates the payout was rejected by the provider
// (e.g., insufficient float balance, invalid recipient phone number).
//
// # Concurrency Safety
//
// All [Disburser] implementations must be safe for concurrent use across multiple goroutines.
type Disburser interface {
	// Disburse transfers funds from the business account to a recipient.
	Disburse(ctx context.Context, req *DisbursementRequest) (*DisbursementResponse, error)
}

// WebhookHandler defines the capability to parse, authenticate, and unpack incoming
// asynchronous notifications (IPNs or webhooks) from a payment provider into a normalized [Event].
//
// Providers that send asynchronous webhook notifications implement this interface.
// It is typically utilized by [webhook.Receiver] to verify incoming HTTP callback requests:
//
//	if wh, ok := gw.(paykit.WebhookHandler); ok {
//	    event, err := wh.ParseAndVerify(req)
//	    if err != nil {
//	        log.Printf("webhook verification failed: %v", err)
//	    }
//	}
//
// # Request Body Handling
//
// ParseAndVerify reads and consumes the entire HTTP request body (r.Body). Callers that
// need to inspect or process the raw payload again after verification should rely on the returned
// [Event.Raw] field or buffer r.Body before calling ParseAndVerify.
//
// # Concurrency Safety
//
// All [WebhookHandler] implementations must be safe for concurrent use across multiple goroutines.
type WebhookHandler interface {
	// ParseAndVerify validates request authenticity (e.g., HMAC-SHA256 signatures, RSA signatures,
	// or IP whitelists) and deserializes the payload into a normalized [Event].
	//
	// It reads and consumes r.Body. Callers should access [Event.Raw] for subsequent access
	// to the original payload.
	ParseAndVerify(r *http.Request) (*Event, error)
}

// BalanceChecker defines the capability to query the current balance of the merchant's account.
//
// Providers supporting balance inquiries implement this interface. Callers discover
// this capability at runtime using Go type assertions:
//
//	if bc, ok := gw.(paykit.BalanceChecker); ok {
//	    resp, err := bc.CheckBalance(ctx, &paykit.BalanceRequest{Currency: "KES"})
//	    if err == nil && resp.Success {
//	        fmt.Printf("Available: %s\n", resp.AvailableBalance)
//	    }
//	}
//
// # Concurrency Safety
//
// All [BalanceChecker] implementations must be safe for concurrent use across multiple goroutines.
type BalanceChecker interface {
	// CheckBalance retrieves current ledger and available balance information from the provider.
	CheckBalance(ctx context.Context, req *BalanceRequest) (*BalanceResponse, error)
}
