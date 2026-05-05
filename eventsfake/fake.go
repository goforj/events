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

func (b *recordingBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

func (b *recordingBus) WithContext(ctx context.Context) events.API {
	return &boundRecordingBus{
		parent: b,
		inner:  b.inner.WithContext(ctx),
	}
}

func (b *recordingBus) Ready() error {
	return b.inner.Ready()
}

func (b *recordingBus) Publish(event any) error {
	b.mu.Lock()
	b.records = append(b.records, Record{Event: event})
	b.mu.Unlock()
	return b.inner.Publish(event)
}

func (b *recordingBus) Subscribe(handler any) (events.Subscription, error) {
	return b.inner.Subscribe(handler)
}

func (b *recordingBus) Records() []Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Record, len(b.records))
	copy(out, b.records)
	return out
}

func (b *recordingBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = nil
}

func (b *boundRecordingBus) WithContext(ctx context.Context) events.API {
	return &boundRecordingBus{
		parent: b.parent,
		inner:  b.parent.inner.WithContext(ctx),
	}
}

func (b *boundRecordingBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

func (b *boundRecordingBus) Ready() error {
	return b.inner.Ready()
}

func (b *boundRecordingBus) Publish(event any) error {
	b.parent.mu.Lock()
	b.parent.records = append(b.parent.records, Record{Event: event})
	b.parent.mu.Unlock()
	return b.inner.Publish(event)
}

func (b *boundRecordingBus) Subscribe(handler any) (events.Subscription, error) {
	return b.inner.Subscribe(handler)
}
