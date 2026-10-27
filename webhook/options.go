package webhook

import (
	"log/slog"
	"net/http"
	"strings"
)

// DefaultMaxBodyBytes specifies the default maximum incoming webhook payload size (1 MiB).
// This prevents memory exhaustion and denial-of-service attacks from unbounded requests.
const DefaultMaxBodyBytes = 1 << 20

// RouteMatcher extracts the target payment provider identifier from an incoming HTTP request.
// It allows consumers to adapt WebhookReceiver to arbitrary URL structures, headers, or query parameters.
type RouteMatcher func(r *http.Request) string

// Option configures a WebhookReceiver instance during construction.
type Option func(*Receiver)

// WithLogger configures an optional structured logger for recording webhook ingestion,
// authentication failures, and handler errors. When omitted or nil, logging is disabled.
func WithLogger(logger *slog.Logger) Option {
	return func(rcv *Receiver) {
		rcv.logger = logger
	}
}

// WithMaxBodyBytes sets the maximum payload size (in bytes) that WebhookReceiver will read from
// incoming requests. If a request body exceeds this limit, WebhookReceiver aborts reading and returns
// HTTP 400 Bad Request to prevent unbounded memory allocation.
//
// If limit <= 0, DefaultMaxBodyBytes (1 MiB) is used.
func WithMaxBodyBytes(limit int64) Option {
	return func(rcv *Receiver) {
		if limit > 0 {
			rcv.maxBodyBytes = limit
		}
	}
}

// WithPathPrefix specifies an expected URL path prefix to be stripped when resolving provider
// names from incoming request paths (e.g., "/webhooks/" or "/api/v1/callbacks").
//
// Trailing slashes are normalized automatically.
func WithPathPrefix(prefix string) Option {
	return func(rcv *Receiver) {
		rcv.pathPrefix = strings.TrimRight(prefix, "/")
	}
}

// WithRouteMatcher configures a custom provider extraction function. This takes precedence over
// default path-based provider resolution, enabling integration with arbitrary routing designs.
func WithRouteMatcher(matcher RouteMatcher) Option {
	return func(rcv *Receiver) {
		rcv.routeMatcher = matcher
	}
}

// WithAcknowledger registers a custom acknowledgment generator for a specific provider identifier.
// When an incoming webhook from this provider is successfully verified and handled, this acknowledger
// is invoked to write the HTTP response instead of the provider's built-in default format.
func WithAcknowledger(provider string, ack Acknowledger) Option {
	return func(rcv *Receiver) {
		if ack != nil && strings.TrimSpace(provider) != "" {
			normalized := strings.ToLower(strings.TrimSpace(provider))
			rcv.acknowledgers[normalized] = ack
		}
	}
}
