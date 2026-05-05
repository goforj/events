package events

import (
	"context"

	"github.com/goforj/events/eventscore"
)

// API is the root application-facing bus contract.
// @group Core
//
// Example: keep an API-typed bus reference
//
//	api, _ := events.NewSync()
//	var bus events.API = api
//	fmt.Println(bus.Driver())
//	// Output: sync
type API interface {
	// WithContext returns a derived bus handle bound to ctx for subsequent operations.
	// @group Core
	WithContext(ctx context.Context) API
	// Driver reports the active bus backend.
	// @group Core
	//
	// Example: inspect the active backend through the interface
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	fmt.Println(bus.Driver())
	//	// Output: sync
	Driver() eventscore.Driver
	// Ready performs a background-context readiness check.
	// @group Core
	//
	// Example: check readiness through the interface
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	fmt.Println(bus.Ready() == nil)
	//	// Output: true
	Ready() error
	// Publish dispatches an event with the background context.
	// @group Core
	//
	// Example: publish a typed event through the interface
	//
	//	type UserCreated struct {
	//		ID string `json:"id"`
	//	}
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	_, _ = bus.Subscribe(func(event UserCreated) {
	//		fmt.Println(event.ID)
	//	})
	//	_ = bus.Publish(UserCreated{ID: "123"})
	//	// Output: 123
	Publish(event any) error
	// Subscribe registers a typed handler using the background context.
	// @group Core
	//
	// Example: subscribe through the interface
	//
	//	type UserCreated struct {
	//		ID string `json:"id"`
	//	}
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	sub, _ := bus.Subscribe(func(ctx context.Context, event UserCreated) error {
	//		_ = ctx
	//		_ = event
	//		return nil
	//	})
	//	defer sub.Close()
	Subscribe(handler any) (Subscription, error)
}

// Subscription releases a subscription when closed.
// @group Subscribe
//
// Example: unsubscribe from a typed event
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	sub, _ := bus.Subscribe(func(event UserCreated) {
//		fmt.Println("received", event.ID)
//	})
//	_ = bus.Publish(UserCreated{ID: "123"})
//	_ = sub.Close()
//	_ = bus.Publish(UserCreated{ID: "456"})
//	// Output: received 123
type Subscription = eventscore.Subscription
