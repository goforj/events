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
	// ReadyContext performs a readiness check with the provided context.
	// @group Core
	//
	// Example: check readiness with a caller context
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	fmt.Println(bus.ReadyContext(context.Background()) == nil)
	//	// Output: true
	ReadyContext(ctx context.Context) error
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
	// PublishContext dispatches an event with the provided context.
	// @group Core
	//
	// Example: publish with a caller context
	//
	//	type UserCreated struct {
	//		ID string `json:"id"`
	//	}
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	_, _ = bus.Subscribe(func(ctx context.Context, event UserCreated) error {
	//		fmt.Println(event.ID, ctx != nil)
	//		return nil
	//	})
	//	_ = bus.PublishContext(context.Background(), UserCreated{ID: "123"})
	//	// Output: 123 true
	PublishContext(ctx context.Context, event any) error
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
	// SubscribeContext registers a typed handler with the provided context.
	// @group Core
	//
	// Example: subscribe with a caller context through the interface
	//
	//	type UserCreated struct {
	//		ID string `json:"id"`
	//	}
	//
	//	api, _ := events.NewSync()
	//	var bus events.API = api
	//	sub, _ := bus.SubscribeContext(context.Background(), func(ctx context.Context, event UserCreated) error {
	//		_ = ctx
	//		_ = event
	//		return nil
	//	})
	//	defer sub.Close()
	SubscribeContext(ctx context.Context, handler any) (Subscription, error)
}

// Subscription releases a subscription when closed.
// @group Subscribe
//
// Example: close a subscription when done
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	sub, _ := bus.Subscribe(func(UserCreated) {})
//	fmt.Println(sub.Close() == nil)
//	// Output: true
type Subscription = eventscore.Subscription
