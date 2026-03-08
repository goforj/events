package events

import (
	"context"

	"github.com/goforj/events/eventscore"
)

// API is the root application-facing bus contract.
type API interface {
	// Driver reports the active bus backend.
	Driver() eventscore.Driver
	// Ready performs a background-context readiness check.
	Ready() error
	// ReadyContext performs a readiness check with the provided context.
	ReadyContext(ctx context.Context) error
	// Publish dispatches an event with the background context.
	Publish(event any) error
	// PublishContext dispatches an event with the provided context.
	PublishContext(ctx context.Context, event any) error
	// Subscribe registers a typed handler using the background context.
	Subscribe(handler any) (Subscription, error)
	// SubscribeContext registers a typed handler with the provided context.
	SubscribeContext(ctx context.Context, handler any) (Subscription, error)
}

// Subscription releases a subscription when closed.
type Subscription = eventscore.Subscription
