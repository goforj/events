package events

import "testing"

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

func BenchmarkResolveTopic(b *testing.B) {
	event := userCreated{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := resolveTopic(event); err != nil {
			b.Fatalf("resolveTopic returned error: %v", err)
		}
	}
}

func BenchmarkNewRegisteredHandler(b *testing.B) {
	handler := func(userCreated) error { return nil }
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := newRegisteredHandler(handler); err != nil {
			b.Fatalf("newRegisteredHandler returned error: %v", err)
		}
	}
}
