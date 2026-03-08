package eventstest

import (
	"context"
	"errors"
	"testing"

	"github.com/goforj/events"
)

// Factory constructs a bus for the shared contract suite.
type Factory func(testing.TB) events.API

type contractEvent struct{}
type overrideEvent struct {
	ID string `json:"id"`
}

func (overrideEvent) Topic() string { return "contracts.override" }

// RunBusContract runs a small backend-agnostic contract suite.
func RunBusContract(t *testing.T, factory Factory) {
	t.Helper()

	t.Run("ready", func(t *testing.T) {
		bus := factory(t)
		if err := bus.Ready(); err != nil {
			t.Fatalf("Ready returned error: %v", err)
		}
	})

	t.Run("publish_subscribe", func(t *testing.T) {
		bus := factory(t)
		called := false
		sub, err := bus.Subscribe(func(ctx context.Context, event contractEvent) error {
			if ctx == nil {
				t.Fatal("expected non-nil context")
			}
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = sub.Close() })
		if err := bus.Publish(contractEvent{}); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if !called {
			t.Fatal("expected handler to be called")
		}
	})

	t.Run("unsubscribe", func(t *testing.T) {
		bus := factory(t)
		calls := 0
		sub, err := bus.Subscribe(func(contractEvent) {
			calls++
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		if err := sub.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
		if err := bus.Publish(contractEvent{}); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if calls != 0 {
			t.Fatalf("calls = %d, want 0", calls)
		}
	})

	t.Run("handler_error", func(t *testing.T) {
		bus := factory(t)
		want := errors.New("boom")
		_, err := bus.Subscribe(func(contractEvent) error {
			return want
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		if err := bus.Publish(contractEvent{}); !errors.Is(err, want) {
			t.Fatalf("Publish error = %v, want %v", err, want)
		}
	})

	t.Run("topic_override", func(t *testing.T) {
		bus := factory(t)
		done := make(chan overrideEvent, 1)
		sub, err := bus.Subscribe(func(event overrideEvent) {
			done <- event
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = sub.Close() })
		if err := bus.Publish(overrideEvent{ID: "override"}); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		select {
		case event := <-done:
			if event.ID != "override" {
				t.Fatalf("event ID = %q, want %q", event.ID, "override")
			}
		default:
			t.Fatal("expected override-routed event delivery")
		}
	})
}

// RunNullBusContract runs the shared contract suite for the drop-only null bus.
func RunNullBusContract(t *testing.T, factory Factory) {
	t.Helper()

	t.Run("ready", func(t *testing.T) {
		bus := factory(t)
		if err := bus.Ready(); err != nil {
			t.Fatalf("Ready returned error: %v", err)
		}
	})

	t.Run("publish_is_noop", func(t *testing.T) {
		bus := factory(t)
		called := false
		sub, err := bus.Subscribe(func(context.Context, contractEvent) error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("Subscribe returned error: %v", err)
		}
		t.Cleanup(func() { _ = sub.Close() })
		if err := bus.Publish(contractEvent{}); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if called {
			t.Fatal("expected null bus to skip handler delivery")
		}
	})
}
