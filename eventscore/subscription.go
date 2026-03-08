package eventscore

import "context"

// Subscription releases a subscription when closed.
type Subscription interface {
	Close() error
}

// MessageHandler receives low-level transport messages.
type MessageHandler func(context.Context, Message) error

// DriverAPI defines the transport-facing driver contract.
type DriverAPI interface {
	Driver() Driver
	Ready(context.Context) error
	PublishContext(context.Context, Message) error
	SubscribeContext(context.Context, string, MessageHandler) (Subscription, error)
}
