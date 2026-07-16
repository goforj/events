package eventsfake

import (
	"context"
	"testing"

	"github.com/goforj/events"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/eventstest"
)

// TestFakeBusContract verifies the reusable fake satisfies the shared bus contract.
func TestFakeBusContract(t *testing.T) {
	eventstest.RunBusContract(t, func(testing.TB) events.API {
		return New().Bus()
	})
}

// TestFakeBusDriverAndSubscribeContext verifies fake driver identity and context subscription behavior.
func TestFakeBusDriverAndSubscribeContext(t *testing.T) {
	fake := New()
	if got := fake.Bus().Driver(); got != eventscore.DriverSync {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSync)
	}
	sub, err := fake.Bus().WithContext(context.Background()).Subscribe(func(fakeEvent) {})
	if err != nil {
		t.Fatalf("WithContext(...).Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// TestFakeBusReadyAndSubscribe verifies readiness and synchronous handler delivery.
func TestFakeBusReadyAndSubscribe(t *testing.T) {
	fake := New()
	if err := fake.Bus().Ready(); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := fake.Bus().WithContext(context.Background()).Ready(); err != nil {
		t.Fatalf("WithContext(...).Ready returned error: %v", err)
	}
	sub, err := fake.Bus().Subscribe(func(fakeEvent) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// TestFakeBusPublishContextAndRecords verifies context publishes are both delivered and recorded.
func TestFakeBusPublishContextAndRecords(t *testing.T) {
	fake := New()
	if err := fake.Bus().WithContext(context.Background()).Publish(fakeEvent{}); err != nil {
		t.Fatalf("WithContext(...).Publish returned error: %v", err)
	}
	records := fake.Records()
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
}
