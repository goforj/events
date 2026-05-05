package events

import (
	"context"
	"testing"
)

func TestNewFakeReadySubscribeReset(t *testing.T) {
	fake := NewFake()
	if err := fake.Bus().Ready(); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := fake.Bus().WithContext(context.Background()).Ready(); err != nil {
		t.Fatalf("WithContext(...).Ready returned error: %v", err)
	}
	sub, err := fake.Bus().Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	fake.Reset()
	if got := fake.Count(); got != 0 {
		t.Fatalf("Count = %d, want 0", got)
	}
}

func TestNewFakeSubscribeContextAndDriver(t *testing.T) {
	fake := NewFake()
	if got := fake.Bus().Driver(); got != fake.bus.Driver() {
		t.Fatalf("Driver = %q, want %q", got, fake.bus.Driver())
	}
	sub, err := fake.Bus().WithContext(context.Background()).Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("WithContext(...).Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
