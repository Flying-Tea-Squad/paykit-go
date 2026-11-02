package webhook_test

import (
	"context"
	"fmt"
	"net/http"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/webhook"
)

// ExampleReceiver demonstrates standard net/http integration.
func ExampleReceiver() {
	// 1. Initialize provider webhook handlers (or resolve from paykit.Get)
	routes := map[string]paykit.WebhookHandler{
		"mpesa": &mockGatewayWebhookHandler{provider: "mpesa"},
	}

	// 2. Define the application event handler
	eventHandler := func(ctx context.Context, evt *paykit.Event) error {
		fmt.Printf("Received %s event for transaction %s\n", evt.Provider, evt.TransactionID)
		return nil
	}

	// 3. Instantiate the WebhookReceiver
	receiver := webhook.NewReceiver(routes, eventHandler)

	// 4. Mount on standard net/http ServeMux
	mux := http.NewServeMux()
	mux.Handle("/webhooks/", receiver)

	// Output:
}

// ExampleReceiver_options demonstrates receiver construction with functional options.
func ExampleReceiver_options() {
	routes := map[string]paykit.WebhookHandler{
		"mpesa":  &mockGatewayWebhookHandler{provider: "mpesa"},
		"airtel": &mockGatewayWebhookHandler{provider: "airtel"},
	}

	eventHandler := func(ctx context.Context, evt *paykit.Event) error {
		return nil
	}

	// Configure receiver with path prefix and body limit
	receiver := webhook.NewReceiver(
		routes,
		eventHandler,
		webhook.WithPathPrefix("/api/v1/callbacks"),
		webhook.WithMaxBodyBytes(512*1024), // 512 KiB limit
	)

	_ = receiver
	// Output:
}
