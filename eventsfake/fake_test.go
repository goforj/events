package eventsfake

import (
	"testing"
)

type fakeEvent struct{}

func TestFakeRecordsPublishes(t *testing.T) {
	fake := New()
	if err := fake.Bus().Publish(fakeEvent{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	fake.AssertCount(t, 1)
}

func TestFakeReset(t *testing.T) {
	fake := New()
	if err := fake.Bus().Publish(fakeEvent{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	fake.Reset()
	fake.AssertNothingPublished(t)
}
