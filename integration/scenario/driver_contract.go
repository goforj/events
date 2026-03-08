package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/goforj/events"
	"github.com/goforj/events/eventscore"
)

type driverEvent struct {
	ID string `json:"id"`
}

type overrideDriverEvent struct {
	ID string `json:"id"`
}

func (overrideDriverEvent) Topic() string { return "integration.override" }

// Factory constructs a transport-backed driver for integration scenarios.
type Factory func(testing.TB, context.Context) eventscore.DriverAPI

// RunDriverContract runs the shared integration scenarios for a distributed driver.
func RunDriverContract(t *testing.T, factory Factory) {
	t.Helper()

	t.Run("ready", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		driver := factory(t, ctx)
		if err := driver.Ready(ctx); err != nil {
			t.Fatalf("Ready returned error: %v", err)
		}
	})

	t.Run("publish_subscribe", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		driver := factory(t, ctx)

		bus, err := events.New(events.Config{Transport: driver})
		if err != nil {
			t.Fatalf("events.New returned error: %v", err)
		}

		done := make(chan driverEvent, 1)
		sub, err := bus.Subscribe(func(event driverEvent) {
			done <- event
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = sub.Close() })

		if err := bus.PublishContext(ctx, driverEvent{ID: "123"}); err != nil {
			t.Fatalf("PublishContext returned error: %v", err)
		}

		select {
		case event := <-done:
			if event.ID != "123" {
				t.Fatalf("event ID = %q, want %q", event.ID, "123")
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for distributed delivery")
		}
	})

	t.Run("unsubscribe", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		driver := factory(t, ctx)

		bus, err := events.New(events.Config{Transport: driver})
		if err != nil {
			t.Fatalf("events.New returned error: %v", err)
		}

		delivered := make(chan struct{}, 1)
		sub, err := bus.Subscribe(func(driverEvent) {
			delivered <- struct{}{}
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		if err := sub.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}

		if err := bus.PublishContext(ctx, driverEvent{ID: "456"}); err != nil {
			t.Fatalf("PublishContext returned error: %v", err)
		}

		select {
		case <-delivered:
			t.Fatal("received message after unsubscribe")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("fanout_subscriptions", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		driver := factory(t, ctx)

		bus, err := events.New(events.Config{Transport: driver})
		if err != nil {
			t.Fatalf("events.New returned error: %v", err)
		}

		firstDone := make(chan driverEvent, 1)
		secondDone := make(chan driverEvent, 1)

		firstSub, err := bus.Subscribe(func(event driverEvent) {
			firstDone <- event
		})
		if err != nil {
			t.Fatalf("first Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = firstSub.Close() })

		secondSub, err := bus.Subscribe(func(event driverEvent) {
			secondDone <- event
		})
		if err != nil {
			t.Fatalf("second Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = secondSub.Close() })

		if err := bus.PublishContext(ctx, driverEvent{ID: "fanout"}); err != nil {
			t.Fatalf("PublishContext returned error: %v", err)
		}

		for name, done := range map[string]chan driverEvent{
			"first":  firstDone,
			"second": secondDone,
		} {
			select {
			case event := <-done:
				if event.ID != "fanout" {
					t.Fatalf("%s subscriber event ID = %q, want %q", name, event.ID, "fanout")
				}
			case <-ctx.Done():
				t.Fatalf("timed out waiting for %s fan-out delivery", name)
			}
		}
	})

	t.Run("topic_override", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		driver := factory(t, ctx)

		bus, err := events.New(events.Config{Transport: driver})
		if err != nil {
			t.Fatalf("events.New returned error: %v", err)
		}

		done := make(chan overrideDriverEvent, 1)
		sub, err := bus.Subscribe(func(event overrideDriverEvent) {
			done <- event
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = sub.Close() })

		if err := bus.PublishContext(ctx, overrideDriverEvent{ID: "override"}); err != nil {
			t.Fatalf("PublishContext returned error: %v", err)
		}

		select {
		case event := <-done:
			if event.ID != "override" {
				t.Fatalf("event ID = %q, want %q", event.ID, "override")
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for override-routed distributed delivery")
		}
	})
}
