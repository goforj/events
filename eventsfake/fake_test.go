package eventsfake

import (
	"testing"
)

type fakeEvent struct{}

// TestFakeRecordsPublishes verifies the reusable fake captures published events.
func TestFakeRecordsPublishes(t *testing.T) {
	fake := New()
	if err := fake.Bus().Publish(fakeEvent{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	fake.AssertCount(t, 1)
}

// TestFakeReset verifies recorded events are cleared without replacing the bus.
func TestFakeReset(t *testing.T) {
	fake := New()
	if err := fake.Bus().Publish(fakeEvent{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	fake.Reset()
	fake.AssertNothingPublished(t)
}
