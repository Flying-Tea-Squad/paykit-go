// Package webhook provides an HTTP handler for ingesting, cryptographically verifying,
// routing, and acknowledging incoming payment provider webhooks (IPNs/callbacks).
//
// # Overview
//
// When payment operations (such as mobile money STK pushes or disbursements) settle
// asynchronously, providers send HTTP callback notifications to the merchant server.
// [Receiver] is an [net/http.Handler] that encapsulates:
//
//   - Request payload reading and body size limiting ([WithMaxBodyBytes]).
//   - Provider route dispatch based on path, query parameter, or custom headers.
//   - Cryptographic signature and payload verification delegated to provider-specific
//     [paykit.WebhookHandler] implementations.
//   - Safe event dispatching to application logic via [EventHandler].
//   - Provider-compliant HTTP acknowledgment serialization via [Acknowledger].
//
// # Mounting Receiver
//
// [Receiver] implements [net/http.Handler] and can be mounted directly on standard Go
// [net/http.ServeMux], as well as routers like Chi, Gin, or Echo:
//
//	routes := map[string]paykit.WebhookHandler{
//	    "mpesa":   mpesaGateway,
//	    "airtel":  airtelGateway,
//	    "pesapal": pesapalGateway,
//	}
//
//	eventHandler := func(ctx context.Context, evt *paykit.Event) error {
//	    log.Printf("Received %s event %s: status=%s", evt.Provider, evt.TransactionID, evt.Status)
//	    return nil
//	}
//
//	receiver := webhook.NewReceiver(routes, eventHandler)
//	http.Handle("/webhooks/", receiver)
//
// In this configuration, requests to `/webhooks/mpesa` are routed to the M-Pesa handler,
// `/webhooks/airtel` to Airtel, and `/webhooks/pesapal` to Pesapal.
//
// # Error Handling and Retries
//
// If the application's [EventHandler] returns a non-nil error, [Receiver] aborts processing
// and responds with HTTP 500 Internal Server Error. This signals the sending payment provider
// that delivery failed, triggering the provider's automated retry schedule.
//
// # Concurrency
//
// [Receiver] is safe for concurrent use across multiple goroutines.
package webhook
