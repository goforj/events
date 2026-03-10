package scenario

import (
	"context"
	"fmt"
	"os"
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

const (
	driverSetupTimeout    = 20 * time.Second
	driverScenarioTimeout = 10 * time.Second
	unsubscribeWait       = 150 * time.Millisecond
)

// Factory constructs a transport-backed driver for integration scenarios.
type Factory func(testing.TB, context.Context) eventscore.DriverAPI

// RunDriverContract runs the shared integration scenarios for a distributed driver.
func RunDriverContract(t *testing.T, factory Factory) {
	t.Helper()

	driverCtx, cancelDriver := context.WithTimeout(context.Background(), driverSetupTimeout)
	defer cancelDriver()

	progressf("creating shared driver fixture")
	driver := factory(t, driverCtx)
	progressf("shared driver fixture ready")

	t.Run("ready", func(t *testing.T) {
		progressf("checking driver readiness")
		ctx, cancel := context.WithTimeout(context.Background(), driverScenarioTimeout)
		defer cancel()

		if err := driver.Ready(ctx); err != nil {
			t.Fatalf("Ready returned error: %v", err)
		}
	})

	t.Run("publish_subscribe", func(t *testing.T) {
		progressf("running publish/subscribe scenario")
		ctx, cancel := context.WithTimeout(context.Background(), driverScenarioTimeout)
		defer cancel()

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
		progressf("running unsubscribe scenario")
		ctx, cancel := context.WithTimeout(context.Background(), driverScenarioTimeout)
		defer cancel()

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
		case <-time.After(unsubscribeWait):
		}
	})

	t.Run("fanout_subscriptions", func(t *testing.T) {
		progressf("running fan-out scenario")
		ctx, cancel := context.WithTimeout(context.Background(), driverScenarioTimeout)
		defer cancel()

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
		progressf("running topic override scenario")
		ctx, cancel := context.WithTimeout(context.Background(), driverScenarioTimeout)
		defer cancel()

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

func progressf(format string, args ...any) {
	if testing.Verbose() {
		fmt.Fprintf(os.Stderr, "[integration] "+format+"\n", args...)
	}
}
