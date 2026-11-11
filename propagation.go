package events

import (
	"context"
	"encoding/json"
	"maps"
	"unicode/utf8"
)

// ContextPropagator transfers explicitly selected context values through message headers.
// Implementations must be safe for concurrent use and preserve receiver cancellation and deadlines.
// Carriers are limited to 10 fields and 8 KiB of both raw key/value bytes and JSON-encoded bytes.
// Oversized carriers and invalid UTF-8 are ignored without rejecting the event payload.
// @group Options
type ContextPropagator interface {
	// Inject returns metadata to snapshot before publishing. An empty map leaves the wire payload unchanged.
	Inject(context.Context) map[string]string
	// Extract restores message metadata onto the receiver's context for one handler invocation.
	Extract(context.Context, map[string]string) context.Context
}

// WithContextPropagator enables optional message context propagation without adding a tracing dependency.
// @group Options
func WithContextPropagator(propagator ContextPropagator) Option {
	return func(options *options) { options.propagator = propagator }
}

// boundedContextCarrier snapshots optional metadata only when it fits the shared carrier budget.
func boundedContextCarrier(headers map[string]string) map[string]string {
	if !validContextCarrier(headers) {
		return nil
	}
	return maps.Clone(headers)
}

// validContextCarrier checks the shared budget before optional metadata reaches a transport or propagator.
func validContextCarrier(headers map[string]string) bool {
	if len(headers) == 0 || len(headers) > 10 {
		return false
	}
	size := 0
	for key, value := range headers {
		size += len(key) + len(value)
		if size > 8192 || !utf8.ValidString(key) || !utf8.ValidString(value) {
			return false
		}
	}
	// Each byte needs at most six JSON bytes. Small carriers fit even with worst-case escaping,
	// avoiding serialization solely to validate the common tracing-header path.
	if size*6+len(headers)*6+1 <= 8192 {
		return true
	}
	// Larger carriers need an exact measurement to preserve the inclusive byte limit.
	encoded, _ := json.Marshal(headers)
	return len(encoded) <= 8192
}
