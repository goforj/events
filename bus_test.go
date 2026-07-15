package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/goforj/events/eventscore"
)

// TestNewDefaultsToSyncDriver verifies an unconfigured bus dispatches synchronously.
func TestNewDefaultsToSyncDriver(t *testing.T) {
	bus, err := New(Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if got := bus.Driver(); got != eventscore.DriverSync {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSync)
	}
	if bus.codec == nil {
		t.Fatal("expected default codec")
	}
}

// TestWithContextSharesConcurrentBusState verifies facades keep context local while sharing one synchronization domain.
func TestWithContextSharesConcurrentBusState(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	derived, ok := bus.WithContext(context.Background()).(*Bus)
	if !ok {
		t.Fatalf("WithContext returned %T, want *Bus", bus.WithContext(context.Background()))
	}
	if derived.busState != bus.busState {
		t.Fatal("WithContext did not preserve shared bus state")
	}

	var deliveries atomic.Int64
	permanent, err := bus.Subscribe(func(userCreated) {
		deliveries.Add(1)
	})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	t.Cleanup(func() { _ = permanent.Close() })

	const publishers = 4
	const publishesPerWorker = 200
	var wg sync.WaitGroup
	for worker := range publishers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var publisher API = bus
			if worker%2 == 1 {
				publisher = derived
			}
			for range publishesPerWorker {
				if err := publisher.Publish(userCreated{}); err != nil {
					t.Errorf("Publish returned error: %v", err)
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range publishesPerWorker {
			var registrar API = bus
			if i%2 == 1 {
				registrar = derived
			}
			sub, subscribeErr := registrar.Subscribe(func(userCreated) {})
			if subscribeErr != nil {
				t.Errorf("Subscribe returned error: %v", subscribeErr)
				return
			}
			if closeErr := sub.Close(); closeErr != nil {
				t.Errorf("Close returned error: %v", closeErr)
				return
			}
		}
	}()
	wg.Wait()

	if got, want := deliveries.Load(), int64(publishers*publishesPerWorker); got != want {
		t.Fatalf("deliveries = %d, want %d", got, want)
	}
}

// TestNewRejectsTransportOnlyDriverWithoutTransport prevents silent sync behavior under a misleading driver name.
func TestNewRejectsTransportOnlyDriverWithoutTransport(t *testing.T) {
	_, err := New(Config{Driver: eventscore.DriverNATS})
	if !errors.Is(err, ErrUnsupportedDriver) {
		t.Fatalf("New error = %v, want %v", err, ErrUnsupportedDriver)
	}
}

// TestNewRejectsTypedNilCollaborators verifies interface wrapping cannot defer wiring failures to first use.
func TestNewRejectsTypedNilCollaborators(t *testing.T) {
	var transport *fakeTransport
	if _, err := New(Config{Transport: transport}); !errors.Is(err, ErrNilTransport) {
		t.Fatalf("New typed-nil transport error = %v, want %v", err, ErrNilTransport)
	}

	var codec *codecFailure
	if _, err := New(Config{Codec: codec}); !errors.Is(err, ErrNilCodec) {
		t.Fatalf("New typed-nil codec error = %v, want %v", err, ErrNilCodec)
	}
	if _, err := NewSync(WithCodec(codec)); !errors.Is(err, ErrNilCodec) {
		t.Fatalf("NewSync typed-nil codec option error = %v, want %v", err, ErrNilCodec)
	}
}

// TestNewSync verifies the explicit synchronous constructor returns a usable bus.
func TestNewSync(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	if got := bus.Driver(); got != eventscore.DriverSync {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSync)
	}
}

// TestNewNull verifies the null constructor accepts publishes without delivery.
func TestNewNull(t *testing.T) {
	bus, err := NewNull()
	if err != nil {
		t.Fatalf("NewNull returned error: %v", err)
	}
	if got := bus.Driver(); got != eventscore.DriverNull {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverNull)
	}
}

// TestReadyUsesBackgroundContext verifies the convenience readiness probe supplies a non-nil context.
func TestReadyUsesBackgroundContext(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	if err := bus.Ready(); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := bus.WithContext(nil).Ready(); err != nil {
		t.Fatalf("WithContext(nil).Ready returned error: %v", err)
	}
}
