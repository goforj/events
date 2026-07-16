package eventscore

import "context"

// Subscription releases a subscription when closed.
type Subscription interface {
	// Close releases the subscription and stops future delivery when the backend permits it.
	Close() error
}

// MessageHandler receives low-level transport messages.
type MessageHandler func(context.Context, Message) error

// DriverAPI defines the transport-facing driver contract.
type DriverAPI interface {
	// Driver reports the backend kind.
	Driver() Driver
	// Ready verifies backend connectivity with the caller context.
	Ready(context.Context) error
	// PublishContext publishes one transport message with the caller context.
	PublishContext(context.Context, Message) error
	// SubscribeContext registers a low-level topic handler with the caller context.
	SubscribeContext(context.Context, string, MessageHandler) (Subscription, error)
}
