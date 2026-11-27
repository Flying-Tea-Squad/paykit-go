// Package paykit provides a unified, capability-based Go SDK for integrating
// African payment providers such as M-Pesa, Airtel Money, and Pesapal.
//
// PayKit is designed around small, focused capability interfaces, strong monetary
// type safety using minor units, explicit error handling semantics, and deterministic
// testing support.
//
// # Capability-Based Architecture
//
// Rather than forcing all payment gateways into a monolithic interface where many
// operations might fail with unsupported feature errors, PayKit defines granular,
// composable capability interfaces:
//
//   - [Gateway]: Primary payment collection interface implemented by all provider adapters.
//     Provides customer payment collection ([Gateway.Charge]), transaction status queries
//     ([Gateway.QueryStatus]), and reversals ([Gateway.Refund]).
//   - [Disburser]: Implemented by providers supporting outbound payouts and B2C mobile transfers
//     ([Disburser.Disburse]).
//   - [WebhookHandler]: Implemented by providers capable of parsing and cryptographically
//     verifying inbound asynchronous notifications ([WebhookHandler.ParseAndVerify]).
//   - [BalanceChecker]: Implemented by providers supporting merchant ledger and available
//     balance queries ([BalanceChecker.CheckBalance]).
//
// Callers discover optional capabilities at runtime using standard Go type assertions:
//
//	gw, err := paykit.Get("mpesa")
//	if err != nil {
//	    log.Fatalf("failed to get gateway: %v", err)
//	}
//
//	if disburser, ok := gw.(paykit.Disburser); ok {
//	    resp, err := disburser.Disburse(ctx, disburseReq)
//	    if err != nil {
//	        log.Printf("disbursement error: %v", err)
//	    }
//	    _ = resp
//	}
//
// # Monetary Representation
//
// All monetary values in PayKit are represented by the immutable [Money] value type
// in integer minor units (cents) alongside an ISO-4217 currency code (e.g., "KES", "USD").
// Using minor units eliminates floating-point rounding errors in financial transactions
// and ledger accounting:
//
//	// Create KES 100.50 (stored as 10050 minor units / cents)
//	m1 := paykit.NewKES(100.50)
//
//	// Create USD 25.00 from explicit minor units
//	m2 := paykit.NewMoney(2500, "USD")
//
// [Money] provides arithmetic operations ([Money.Add], [Money.Sub], [Money.Mul], [Money.Cmp])
// that enforce currency matching and panic on cross-currency operations, preventing currency
// mismatch bugs.
//
// # Phone Number Normalization
//
// African mobile money networks require standardized MSISDN formats. PayKit provides
// [FormatKenyanPhone] to validate and normalize Kenyan mobile phone numbers across all
// network operators (Safaricom, Airtel, Telkom) to the 12-digit E.164 format:
//
//	formatted, err := paykit.FormatKenyanPhone("0712345678")
//	// formatted == "254712345678"
//
// Supported input formats include local formats with or without leading zeros ("07...", "7..."),
// international formats ("254...", "+254..."), and formatted strings with spaces or hyphens.
//
// # Dual-Mode Usage: Direct Constructors vs. Dynamic Registry
//
// PayKit supports two integration workflows depending on application architecture:
//
// 1. Direct Provider Construction: Instantiate provider clients directly from their
// respective packages (e.g., mpesa.New, airtel.New, bogus.New). This approach offers
// full compile-time type safety and explicit configuration:
//
//	client := mpesa.New(mpesa.Config{
//	    ConsumerKey:    "your-key",
//	    ConsumerSecret: "your-secret",
//	    PassKey:        "your-passkey",
//	    ShortCode:      "174379",
//	    Environment:    mpesa.Sandbox,
//	})
//
// 2. Dynamic Driver Registry: Register and resolve gateways dynamically by provider name using
// [Register] and [Get], modeled after the standard library database/sql package.
// Providers register themselves with [Register], or applications can import driver packages
// for blank side-effect registration:
//
//	import _ "github.com/Flying-Tea-Squad/paykit-go/driver"
//
//	gw, err := paykit.Get("mpesa")
//
// # Synchronous vs. Asynchronous Lifecycle Semantics
//
// Most African mobile money payment flows (such as M-Pesa STK Push / USSD Push) are
// asynchronous by nature. Calling [Gateway.Charge] initiates the customer prompt:
//
//   - An initial successful charge call returns a [ChargeResponse] with [TransactionStatus]
//     set to [StatusPendingAction] or [StatusProcessing]. This confirms the push request
//     was accepted by the telco gateway, NOT that funds were settled.
//   - Final settlement is communicated asynchronously via provider callbacks (webhooks),
//     delivering a terminal state: [StatusSuccess], [StatusFailed], [StatusCancelled],
//     or [StatusTimeout].
//   - Callers should use the [webhook] package to process callbacks, and use
//     [TransactionStatus.IsTerminal] to verify whether a transaction has reached its final state.
//
// # Error Semantics: Transport Errors vs. Provider Rejections
//
// PayKit strictly differentiates between execution/transport failures and provider-level
// business rejections:
//
//   - Go error (err != nil): Indicates transport or infrastructure failure—such as
//     network timeouts, DNS errors, context cancellations, or invalid parameters.
//     The response struct may be nil.
//   - Provider Rejection (err == nil && !resp.Success): Indicates the HTTP exchange succeeded,
//     but the payment provider rejected the operation (e.g., insufficient merchant balance,
//     invalid shortcode, duplicate request). The response struct is populated with
//     human-readable details in [ChargeResponse.Message], normalized status, and
//     provider-specific details in [ChargeResponse.Metadata] or [ChargeResponse.Raw].
//   - Provider error codes can be normalized to standardized PayKit error constants
//     using [MapGatewayErrorCode].
//
// # Concurrency Safety
//
// All PayKit capability interface implementations ([Gateway], [Disburser], [WebhookHandler],
// [BalanceChecker]), as well as the [webhook.Receiver] and [Money] value objects, are safe
// for concurrent use across multiple goroutines.
package paykit
