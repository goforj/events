package events

import "context"

// ContextPropagator transfers explicitly selected context values through message headers.
// Implementations must be safe for concurrent use and preserve receiver cancellation and deadlines.
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
