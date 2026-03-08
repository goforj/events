package events

import (
	"context"
	"testing"
)

func TestNewFakeRecordsPublishes(t *testing.T) {
	fake := NewFake()
	if err := fake.Bus().Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if got := fake.Count(); got != 1 {
		t.Fatalf("Count = %d, want 1", got)
	}
}

func TestNewFakeSupportsReadyAndContextPublish(t *testing.T) {
	fake := NewFake()
	if err := fake.Bus().Ready(); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := fake.Bus().PublishContext(context.Background(), userCreated{}); err != nil {
		t.Fatalf("PublishContext returned error: %v", err)
	}
	records := fake.Records()
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
}
