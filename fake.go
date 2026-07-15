package events

import (
	"context"
	"sync"

	"github.com/goforj/events/eventscore"
)

// Record captures one published event observed by a Fake bus.
// @group Testing
//
// Example: inspect a recorded event
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	record := events.Record{Event: UserCreated{ID: "123"}}
//	fmt.Printf("%T\n", record.Event)
//	// Output: main.UserCreated
type Record struct {
	Event any
}

// Fake provides a root-package testing helper that records published events.
// @group Testing
//
// Example: keep a fake for assertions in tests
//
//	fake := events.NewFake()
//	fmt.Println(fake.Count())
//	// Output: 0
type Fake struct {
	bus *fakeBus
}

// NewFake creates a new fake event harness backed by the root sync bus.
// @group Testing
//
// Example: construct a recording fake
//
//	fake := events.NewFake()
//	fmt.Println(fake.Count())
//	// Output: 0
func NewFake() *Fake {
	bus, err := NewSync()
	if err != nil {
		panic(err)
	}
	return &Fake{bus: &fakeBus{inner: bus}}
}

// Bus returns the wrapped API to inject into code under test.
// @group Testing
//
// Example: inject the fake bus into application code
//
//	fake := events.NewFake()
//	bus := fake.Bus()
//	fmt.Println(bus.Ready() == nil)
//	// Output: true
func (f *Fake) Bus() API {
	return f.bus
}

// Records returns a copy of recorded publishes.
// @group Testing
//
// Example: inspect recorded publishes
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	fake := events.NewFake()
//	_ = fake.Bus().Publish(UserCreated{ID: "123"})
//	fmt.Println(len(fake.Records()))
//	// Output: 1
func (f *Fake) Records() []Record {
	return f.bus.Records()
}

// Reset clears recorded publishes.
// @group Testing
//
// Example: clear recorded publishes
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	fake := events.NewFake()
//	_ = fake.Bus().Publish(UserCreated{ID: "123"})
//	fake.Reset()
//	fmt.Println(fake.Count())
//	// Output: 0
func (f *Fake) Reset() {
	f.bus.Reset()
}

// Count returns the total number of recorded publishes.
// @group Testing
//
// Example: count recorded publishes
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	fake := events.NewFake()
//	_ = fake.Bus().Publish(UserCreated{ID: "123"})
//	fmt.Println(fake.Count())
//	// Output: 1
func (f *Fake) Count() int {
	return len(f.bus.Records())
}

type fakeBus struct {
	inner API

	mu      sync.Mutex
	records []Record
}

type boundFakeBus struct {
	parent *fakeBus
	inner  API
}

// Driver preserves the wrapped backend identity for code exercised through the fake.
func (b *fakeBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

// WithContext keeps context-bound operations attached to the same recording state.
func (b *fakeBus) WithContext(ctx context.Context) API {
	return &boundFakeBus{
		parent: b,
		inner:  b.inner.WithContext(ctx),
	}
}

// Ready delegates readiness so the fake preserves the wrapped bus contract.
func (b *fakeBus) Ready() error {
	return b.inner.Ready()
}

// Publish records the attempt before delegation so failed publishes remain observable in tests.
func (b *fakeBus) Publish(event any) error {
	b.mu.Lock()
	b.records = append(b.records, Record{Event: event})
	b.mu.Unlock()
	return b.inner.Publish(event)
}

// Subscribe delegates handler registration without introducing fake-only delivery semantics.
func (b *fakeBus) Subscribe(handler any) (Subscription, error) {
	return b.inner.Subscribe(handler)
}

// Records returns a snapshot so callers cannot mutate concurrent fake state.
func (b *fakeBus) Records() []Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Record, len(b.records))
	copy(out, b.records)
	return out
}

// Reset clears shared recording state without replacing the wrapped bus.
func (b *fakeBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = nil
}

// WithContext rebinds operations while preserving the parent recorder.
func (b *boundFakeBus) WithContext(ctx context.Context) API {
	return &boundFakeBus{
		parent: b.parent,
		inner:  b.parent.inner.WithContext(ctx),
	}
}

// Driver preserves the wrapped context-bound backend identity.
func (b *boundFakeBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

// Ready delegates through the bound handle so its context remains effective.
func (b *boundFakeBus) Ready() error {
	return b.inner.Ready()
}

// Publish records through the shared parent before context-bound delegation.
func (b *boundFakeBus) Publish(event any) error {
	b.parent.mu.Lock()
	b.parent.records = append(b.parent.records, Record{Event: event})
	b.parent.mu.Unlock()
	return b.inner.Publish(event)
}

// Subscribe delegates through the bound handle so subscription context remains effective.
func (b *boundFakeBus) Subscribe(handler any) (Subscription, error) {
	return b.inner.Subscribe(handler)
}
