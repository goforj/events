package eventsfake

import (
	"context"
	"sync"
	"testing"

	"github.com/goforj/events"
	"github.com/goforj/events/eventscore"
)

// Record captures one published event observed by the fake.
type Record struct {
	Event any
}

// Fake provides an assertion-oriented event bus test harness.
type Fake struct {
	bus *recordingBus
}

// New creates a new fake event harness.
func New() *Fake {
	bus, err := events.NewSync()
	if err != nil {
		panic(err)
	}
	return &Fake{bus: &recordingBus{inner: bus}}
}

// Bus returns the wrapped API to inject into code under test.
func (f *Fake) Bus() events.API {
	return f.bus
}

// Records returns a copy of recorded publishes.
func (f *Fake) Records() []Record {
	return f.bus.Records()
}

// Reset clears recorded publishes.
func (f *Fake) Reset() {
	f.bus.Reset()
}

// Count returns the total number of recorded publishes.
func (f *Fake) Count() int {
	return len(f.bus.Records())
}

// AssertCount verifies the number of recorded publishes.
func (f *Fake) AssertCount(t testing.TB, want int) {
	t.Helper()
	if got := f.Count(); got != want {
		t.Fatalf("publish count = %d, want %d", got, want)
	}
}

// AssertNothingPublished verifies that no publishes were recorded.
func (f *Fake) AssertNothingPublished(t testing.TB) {
	t.Helper()
	f.AssertCount(t, 0)
}

type recordingBus struct {
	inner events.API

	mu      sync.Mutex
	records []Record
}

type boundRecordingBus struct {
	parent *recordingBus
	inner  events.API
}

// Driver preserves the wrapped backend identity for code exercised through the fake.
func (b *recordingBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

// WithContext keeps context-bound operations attached to the same recording state.
func (b *recordingBus) WithContext(ctx context.Context) events.API {
	return &boundRecordingBus{
		parent: b,
		inner:  b.inner.WithContext(ctx),
	}
}

// Ready delegates readiness so the fake preserves the wrapped bus contract.
func (b *recordingBus) Ready() error {
	return b.inner.Ready()
}

// Publish records the attempt before delegation so failed publishes remain observable in tests.
func (b *recordingBus) Publish(event any) error {
	b.mu.Lock()
	b.records = append(b.records, Record{Event: event})
	b.mu.Unlock()
	return b.inner.Publish(event)
}

// Subscribe delegates handler registration without introducing fake-only delivery semantics.
func (b *recordingBus) Subscribe(handler any) (events.Subscription, error) {
	return b.inner.Subscribe(handler)
}

// Records returns a snapshot so callers cannot mutate concurrent fake state.
func (b *recordingBus) Records() []Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Record, len(b.records))
	copy(out, b.records)
	return out
}

// Reset clears shared recording state without replacing the wrapped bus.
func (b *recordingBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = nil
}

// WithContext rebinds operations while preserving the parent recorder.
func (b *boundRecordingBus) WithContext(ctx context.Context) events.API {
	return &boundRecordingBus{
		parent: b.parent,
		inner:  b.parent.inner.WithContext(ctx),
	}
}

// Driver preserves the wrapped context-bound backend identity.
func (b *boundRecordingBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

// Ready delegates through the bound handle so its context remains effective.
func (b *boundRecordingBus) Ready() error {
	return b.inner.Ready()
}

// Publish records through the shared parent before context-bound delegation.
func (b *boundRecordingBus) Publish(event any) error {
	b.parent.mu.Lock()
	b.parent.records = append(b.parent.records, Record{Event: event})
	b.parent.mu.Unlock()
	return b.inner.Publish(event)
}

// Subscribe delegates through the bound handle so subscription context remains effective.
func (b *boundRecordingBus) Subscribe(handler any) (events.Subscription, error) {
	return b.inner.Subscribe(handler)
}
