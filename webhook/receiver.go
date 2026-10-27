package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

// EventHandler defines the application callback invoked upon successful webhook verification.
// If the handler returns a non-nil error, WebhookReceiver aborts processing and returns
// HTTP 500 Internal Server Error to signal the payment provider to retry delivery.
type EventHandler func(ctx context.Context, event *paykit.Event) error

// Receiver is an http.Handler that ingests, validates, routes, and acknowledges incoming
// payment provider webhooks across standard Go HTTP frameworks (net/http, Gin, Chi, Echo).
//
// Concurrency: Receiver is safe for concurrent use across multiple goroutines.
type Receiver struct {
	routes        map[string]paykit.WebhookHandler
	eventHandler  EventHandler
	logger        *slog.Logger
	maxBodyBytes  int64
	pathPrefix    string
	routeMatcher  RouteMatcher
	acknowledgers map[string]Acknowledger
	mu            sync.RWMutex
}

// NewReceiver creates a new WebhookReceiver HTTP handler configured with the given provider routes
// and application event handler callback.
//
// Optional functional configurations (such as WithLogger, WithMaxBodyBytes, WithPathPrefix,
// WithRouteMatcher, or WithAcknowledger) can be passed to customize behavior.
//
// The routes map is defensively copied to prevent external mutation. Provider names are normalized
// to lowercase for case-insensitive URL matching.
func NewReceiver(
	routes map[string]paykit.WebhookHandler,
	eventHandler func(context.Context, *paykit.Event) error,
	opts ...Option,
) *Receiver {
	rcv := &Receiver{
		routes:        make(map[string]paykit.WebhookHandler),
		eventHandler:  eventHandler,
		maxBodyBytes:  DefaultMaxBodyBytes,
		acknowledgers: make(map[string]Acknowledger),
	}

	for k, v := range routes {
		if v != nil && strings.TrimSpace(k) != "" {
			rcv.routes[strings.ToLower(strings.TrimSpace(k))] = v
		}
	}

	for _, opt := range opts {
		if opt != nil {
			opt(rcv)
		}
	}

	return rcv
}

// Register adds or updates a provider webhook handler dynamically.
//
// Side Effects: Register acquires a write lock on the internal routing table.
func (rcv *Receiver) Register(provider string, handler paykit.WebhookHandler) {
	if handler == nil || strings.TrimSpace(provider) == "" {
		return
	}
	rcv.mu.Lock()
	defer rcv.mu.Unlock()
	rcv.routes[strings.ToLower(strings.TrimSpace(provider))] = handler
}

// Handler returns the registered WebhookHandler for a given provider, if present.
//
// Side Effects: Handler acquires a read lock on the internal routing table.
func (rcv *Receiver) Handler(provider string) (paykit.WebhookHandler, bool) {
	rcv.mu.RLock()
	defer rcv.mu.RUnlock()
	h, ok := rcv.routes[strings.ToLower(strings.TrimSpace(provider))]
	return h, ok
}

// errorResponse represents a standardized JSON error payload written on webhook ingestion failures.
type errorResponse struct {
	Error string `json:"error"`
}

// writeErrorResponse writes a JSON-encoded error message with the specified HTTP status code.
func (rcv *Receiver) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

// resolveProvider determines the target payment provider identifier from the incoming request.
func (rcv *Receiver) resolveProvider(r *http.Request) string {
	if rcv.routeMatcher != nil {
		return strings.ToLower(strings.TrimSpace(rcv.routeMatcher(r)))
	}

	urlPath := path.Clean(r.URL.Path)

	// Strip configured prefix if present.
	if rcv.pathPrefix != "" && strings.HasPrefix(urlPath, rcv.pathPrefix) {
		urlPath = strings.TrimPrefix(urlPath, rcv.pathPrefix)
	}

	trimmed := strings.Trim(urlPath, "/")
	if trimmed == "" {
		return ""
	}

	segments := strings.Split(trimmed, "/")

	rcv.mu.RLock()
	defer rcv.mu.RUnlock()

	// 1. Direct match on the last non-empty segment (e.g. /webhooks/mpesa -> mpesa).
	last := strings.ToLower(segments[len(segments)-1])
	if _, ok := rcv.routes[last]; ok {
		return last
	}

	// 2. Search all path segments from right to left for registered provider keys.
	for i := len(segments) - 1; i >= 0; i-- {
		seg := strings.ToLower(segments[i])
		if _, ok := rcv.routes[seg]; ok {
			return seg
		}
	}

	// Fallback to the last segment if no route matches (so 404 includes the candidate name).
	return last
}

// readAndBufferPayload reads incoming request body bytes up to maxBodyBytes and reconstructs
// r.Body and r.GetBody so downstream verifiers and parsers can consume it repeatedly.
func (rcv *Receiver) readAndBufferPayload(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("webhook: request body is nil")
	}

	defer func() { _ = r.Body.Close() }()

	// Read up to limit + 1 bytes to detect payloads exceeding maximum allowance.
	reader := io.LimitReader(r.Body, rcv.maxBodyBytes+1)
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("webhook: failed to read request body: %w", err)
	}

	if int64(len(payload)) > rcv.maxBodyBytes {
		return nil, fmt.Errorf("webhook: request body exceeds maximum limit of %d bytes", rcv.maxBodyBytes)
	}

	// Reconstruct r.Body and r.GetBody for multiple stream reads.
	r.Body = io.NopCloser(bytes.NewReader(payload))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}

	return payload, nil
}

// writeAcknowledgment writes provider-tailored acknowledgment responses to w.
func (rcv *Receiver) writeAcknowledgment(
	w http.ResponseWriter,
	r *http.Request,
	provider string,
	handler paykit.WebhookHandler,
	event *paykit.Event,
) error {
	// 1. If handler itself implements Acknowledger, delegate directly.
	if ack, ok := handler.(Acknowledger); ok {
		return ack.Acknowledge(w, r, event)
	}

	// 2. If custom acknowledger was registered via WithAcknowledger, invoke it.
	rcv.mu.RLock()
	customAck, hasCustom := rcv.acknowledgers[provider]
	rcv.mu.RUnlock()
	if hasCustom && customAck != nil {
		return customAck.Acknowledge(w, r, event)
	}

	// 3. Fallback to built-in provider acknowledgment schemas.
	switch provider {
	case "mpesa", "safaricom":
		return defaultMpesaAcknowledger.Acknowledge(w, r, event)
	case "airtel", "airtelmoney":
		return defaultAirtelAcknowledger.Acknowledge(w, r, event)
	case "pesapal":
		return defaultPesapalAcknowledger.Acknowledge(w, r, event)
	default:
		return defaultGeneralAcknowledger.Acknowledge(w, r, event)
	}
}

// ServeHTTP satisfies the net/http.Handler interface. It executes the complete webhook
// lifecycle: method verification, provider resolution, payload extraction, cryptographic
// verification, application dispatch, and provider acknowledgment generation.
//
// Side Effects:
//   - Fully reads and consumes r.Body, replacing it with a reusable in-memory reader.
//   - Dispatches the verified event to the configured eventHandler.
//   - Writes HTTP status codes, Content-Type headers, and JSON responses to w.
func (rcv *Receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Validate HTTP method. Webhook callbacks must be HTTP POST.
	if r.Method != http.MethodPost {
		if rcv.logger != nil {
			rcv.logger.WarnContext(ctx, "webhook: method not allowed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)
		}
		w.Header().Set("Allow", http.MethodPost)
		rcv.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed: webhooks must be POST requests")
		return
	}

	// 2. Resolve provider route from request.
	provider := rcv.resolveProvider(r)
	if provider == "" {
		if rcv.logger != nil {
			rcv.logger.WarnContext(ctx, "webhook: empty provider route in path",
				slog.String("path", r.URL.Path),
			)
		}
		rcv.writeErrorResponse(w, http.StatusNotFound, "provider route not specified")
		return
	}

	handler, ok := rcv.Handler(provider)
	if !ok || handler == nil {
		if rcv.logger != nil {
			rcv.logger.WarnContext(ctx, "webhook: unregistered provider route",
				slog.String("provider", provider),
				slog.String("path", r.URL.Path),
			)
		}
		rcv.writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("unregistered webhook provider: %s", provider))
		return
	}

	// 3. Extract and buffer raw payload bytes.
	rawPayload, err := rcv.readAndBufferPayload(r)
	if err != nil {
		if rcv.logger != nil {
			rcv.logger.WarnContext(ctx, "webhook: payload read error",
				slog.String("provider", provider),
				slog.String("error", err.Error()),
			)
		}
		rcv.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// 4. Authenticate and parse incoming webhook via provider handler.
	event, err := handler.ParseAndVerify(r)
	if err != nil {
		if rcv.logger != nil {
			rcv.logger.WarnContext(ctx, "webhook: signature verification or parsing failed",
				slog.String("provider", provider),
				slog.String("error", err.Error()),
			)
		}
		rcv.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid webhook payload or signature: %v", err))
		return
	}

	if event == nil {
		if rcv.logger != nil {
			rcv.logger.WarnContext(ctx, "webhook: handler returned nil event",
				slog.String("provider", provider),
			)
		}
		rcv.writeErrorResponse(w, http.StatusBadRequest, "handler returned empty event")
		return
	}

	// Ensure essential metadata is populated if omitted by the provider handler.
	if event.Provider == "" {
		event.Provider = provider
	}
	if len(event.Raw) == 0 {
		event.Raw = json.RawMessage(rawPayload)
	}

	// 5. Dispatch event to the application event handler.
	if rcv.eventHandler != nil {
		if err := rcv.eventHandler(ctx, event); err != nil {
			if rcv.logger != nil {
				rcv.logger.ErrorContext(ctx, "webhook: application event handler error",
					slog.String("provider", provider),
					slog.String("event_type", event.Type),
					slog.String("transaction_id", event.TransactionID),
					slog.String("error", err.Error()),
				)
			}
			rcv.writeErrorResponse(w, http.StatusInternalServerError, "failed to process webhook event")
			return
		}
	}

	// 6. Generate and send provider acknowledgment.
	if err := rcv.writeAcknowledgment(w, r, provider, handler, event); err != nil {
		if rcv.logger != nil {
			rcv.logger.ErrorContext(ctx, "webhook: failed to write provider acknowledgment",
				slog.String("provider", provider),
				slog.String("error", err.Error()),
			)
		}
		return
	}

	if rcv.logger != nil {
		rcv.logger.DebugContext(ctx, "webhook: successfully processed and acknowledged",
			slog.String("provider", provider),
			slog.String("event_type", event.Type),
			slog.String("transaction_id", event.TransactionID),
		)
	}
}
