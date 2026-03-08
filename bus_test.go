package events

import (
	"testing"

	"github.com/goforj/events/eventscore"
)

func TestNewDefaultsToSyncDriver(t *testing.T) {
	bus, err := New(Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if got := bus.Driver(); got != eventscore.DriverSync {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSync)
	}
	if bus.codec == nil {
		t.Fatal("expected default codec")
	}
}

func TestNewSync(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	if got := bus.Driver(); got != eventscore.DriverSync {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSync)
	}
}

func TestNewNull(t *testing.T) {
	bus, err := NewNull()
	if err != nil {
		t.Fatalf("NewNull returned error: %v", err)
	}
	if got := bus.Driver(); got != eventscore.DriverNull {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverNull)
	}
}

func TestReadyUsesBackgroundContext(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	if err := bus.Ready(); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := bus.ReadyContext(nil); err != nil {
		t.Fatalf("ReadyContext returned error: %v", err)
	}
}
