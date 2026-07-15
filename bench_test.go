package events

import "testing"

// BenchmarkPublishNoSubscribers measures synchronous publish overhead without handler dispatch.
func BenchmarkPublishNoSubscribers(b *testing.B) {
	bus, err := NewSync()
	if err != nil {
		b.Fatalf("NewSync returned error: %v", err)
	}
	event := userCreated{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bus.Publish(event); err != nil {
			b.Fatalf("Publish returned error: %v", err)
		}
	}
}

// BenchmarkPublishOneSubscriber measures synchronous dispatch to one no-op handler.
func BenchmarkPublishOneSubscriber(b *testing.B) {
	bus, err := NewSync()
	if err != nil {
		b.Fatalf("NewSync returned error: %v", err)
	}
	_, err = bus.Subscribe(func(userCreated) {})
	if err != nil {
		b.Fatalf("Subscribe returned error: %v", err)
	}
	event := userCreated{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bus.Publish(event); err != nil {
			b.Fatalf("Publish returned error: %v", err)
		}
	}
}

// BenchmarkSyncPublishRoundTrip measures publish plus observable synchronous delivery.
func BenchmarkSyncPublishRoundTrip(b *testing.B) {
	bus, err := NewSync()
	if err != nil {
		b.Fatalf("NewSync returned error: %v", err)
	}
	delivered := make(chan struct{}, 1)
	_, err = bus.Subscribe(func(userCreated) {
		select {
		case delivered <- struct{}{}:
		default:
		}
	})
	if err != nil {
		b.Fatalf("Subscribe returned error: %v", err)
	}
	event := userCreated{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bus.Publish(event); err != nil {
			b.Fatalf("Publish returned error: %v", err)
		}
		select {
		case <-delivered:
		default:
			b.Fatal("expected sync round-trip delivery")
		}
	}
}

// BenchmarkPublishMultipleSubscribers measures fan-out cost across four handlers.
func BenchmarkPublishMultipleSubscribers(b *testing.B) {
	bus, err := NewSync()
	if err != nil {
		b.Fatalf("NewSync returned error: %v", err)
	}
	for i := 0; i < 4; i++ {
		_, err = bus.Subscribe(func(userCreated) {})
		if err != nil {
			b.Fatalf("Subscribe returned error: %v", err)
		}
	}
	event := userCreated{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bus.Publish(event); err != nil {
			b.Fatalf("Publish returned error: %v", err)
		}
	}
}

// BenchmarkResolveTopic measures reflection-based topic derivation for an ordinary event.
func BenchmarkResolveTopic(b *testing.B) {
	event := userCreated{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := resolveTopic(event); err != nil {
			b.Fatalf("resolveTopic returned error: %v", err)
		}
	}
}

// BenchmarkNewRegisteredHandler measures validation and adapter creation for a typed handler.
func BenchmarkNewRegisteredHandler(b *testing.B) {
	handler := func(userCreated) error { return nil }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := newRegisteredHandler(handler); err != nil {
			b.Fatalf("newRegisteredHandler returned error: %v", err)
		}
	}
}
