package eventsfake

import (
	"context"
	"testing"

	"github.com/goforj/events"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/eventstest"
)

func TestFakeBusContract(t *testing.T) {
	eventstest.RunBusContract(t, func(testing.TB) events.API {
		return New().Bus()
	})
}

func TestFakeBusDriverAndSubscribeContext(t *testing.T) {
	fake := New()
	if got := fake.Bus().Driver(); got != eventscore.DriverSync {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSync)
	}
	sub, err := fake.Bus().SubscribeContext(context.Background(), func(fakeEvent) {})
	if err != nil {
		t.Fatalf("SubscribeContext returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestFakeBusReadyAndSubscribe(t *testing.T) {
	fake := New()
	if err := fake.Bus().Ready(); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := fake.Bus().ReadyContext(context.Background()); err != nil {
		t.Fatalf("ReadyContext returned error: %v", err)
	}
	sub, err := fake.Bus().Subscribe(func(fakeEvent) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestFakeBusPublishContextAndRecords(t *testing.T) {
	fake := New()
	if err := fake.Bus().PublishContext(context.Background(), fakeEvent{}); err != nil {
		t.Fatalf("PublishContext returned error: %v", err)
	}
	records := fake.Records()
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
}
