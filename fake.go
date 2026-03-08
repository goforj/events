package events

import (
	"context"
	"sync"

	"github.com/goforj/events/eventscore"
)

// Record captures one published event observed by a Fake bus.
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

func (b *fakeBus) Driver() eventscore.Driver {
	return b.inner.Driver()
}

func (b *fakeBus) Ready() error {
	return b.inner.Ready()
}

func (b *fakeBus) ReadyContext(ctx context.Context) error {
	return b.inner.ReadyContext(ctx)
}

func (b *fakeBus) Publish(event any) error {
	b.mu.Lock()
	b.records = append(b.records, Record{Event: event})
	b.mu.Unlock()
	return b.inner.Publish(event)
}

func (b *fakeBus) PublishContext(ctx context.Context, event any) error {
	b.mu.Lock()
	b.records = append(b.records, Record{Event: event})
	b.mu.Unlock()
	return b.inner.PublishContext(ctx, event)
}

func (b *fakeBus) Subscribe(handler any) (Subscription, error) {
	return b.inner.Subscribe(handler)
}

func (b *fakeBus) SubscribeContext(ctx context.Context, handler any) (Subscription, error) {
	return b.inner.SubscribeContext(ctx, handler)
}

func (b *fakeBus) Records() []Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Record, len(b.records))
	copy(out, b.records)
	return out
}

func (b *fakeBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.records = nil
}
