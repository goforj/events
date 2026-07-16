package events

import (
	"context"
	"errors"
	"testing"
)

type concreteHandlerContext struct {
	context.Context
}

// TestNewRegisteredHandlerRejectsInvalidSignatures verifies handlers require one event argument and an optional error result.
func TestNewRegisteredHandlerRejectsInvalidSignatures(t *testing.T) {
	var nilHandler func(userCreated)
	tests := []any{
		nil,
		nilHandler,
		42,
		func() {},
		func(string, string, string) {},
		func(context.Context, userCreated) string { return "" },
		func(concreteHandlerContext, userCreated) {},
		func(v any) {},
	}
	for _, test := range tests {
		if _, err := newRegisteredHandler(test); err == nil {
			t.Fatalf("expected error for handler %#v", test)
		}
	}
}

// TestPublishDecodesMultiplePointerLevels verifies reflection never changes a valid handler into a runtime panic.
func TestPublishDecodesMultiplePointerLevels(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	called := false
	_, err = bus.Subscribe(func(event **userCreated) {
		if event == nil || *event == nil {
			t.Fatal("expected both pointer levels to be populated")
		}
		called = true
	})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if !called {
		t.Fatal("expected pointer handler to be called")
	}
}

type valueHandlerError struct{}

// Error implements error without requiring a nilable concrete value.
func (valueHandlerError) Error() string { return "value handler error" }

// TestPublishSupportsConcreteValueErrorReturn verifies error extraction does not call IsNil on a struct.
func TestPublishSupportsConcreteValueErrorReturn(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	_, err = bus.Subscribe(func(userCreated) valueHandlerError { return valueHandlerError{} })
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	var got valueHandlerError
	if err := bus.Publish(userCreated{}); !errors.As(err, &got) {
		t.Fatalf("Publish error = %v, want valueHandlerError", err)
	}
}

// TestPublishDispatchesSyncHandler verifies synchronous publishes decode and invoke registered handlers.
func TestPublishDispatchesSyncHandler(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	called := false
	_, err = bus.Subscribe(func(ctx context.Context, event userCreated) error {
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

// TestPublishReturnsFirstHandlerError verifies dispatch stops at the first handler failure.
func TestPublishReturnsFirstHandlerError(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	want := errors.New("boom")
	_, err = bus.Subscribe(func(userCreated) error { return want })
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); !errors.Is(err, want) {
		t.Fatalf("Publish error = %v, want %v", err, want)
	}
}

// TestPublishInvokesHandlersInRegistrationOrder verifies synchronous delivery preserves subscription order.
func TestPublishInvokesHandlersInRegistrationOrder(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	var order []int
	_, err = bus.Subscribe(func(userCreated) { order = append(order, 1) })
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	_, err = bus.Subscribe(func(userCreated) { order = append(order, 2) })
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("handler order = %v, want [1 2]", order)
	}
}

// TestNullBusNoOps verifies every null-bus operation succeeds without invoking handlers.
func TestNullBusNoOps(t *testing.T) {
	bus, err := NewNull()
	if err != nil {
		t.Fatalf("NewNull returned error: %v", err)
	}
	sub, err := bus.Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

type codecFailure struct {
	err error
}

// Marshal returns the configured failure so Publish error propagation can be isolated.
func (c codecFailure) Marshal(any) ([]byte, error) { return nil, c.err }

// Unmarshal returns the configured failure so delivery error propagation can be isolated.
func (c codecFailure) Unmarshal([]byte, any) error { return c.err }

// TestPublishReturnsCodecMarshalError verifies encoding failures prevent transport publication.
func TestPublishReturnsCodecMarshalError(t *testing.T) {
	want := errors.New("marshal failed")
	bus, err := NewSync(WithCodec(codecFailure{err: want}))
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); !errors.Is(err, want) {
		t.Fatalf("Publish error = %v, want %v", err, want)
	}
}

type decodeMismatch struct {
	Count int `json:"count"`
}

// Topic routes the valid payload to the shared decode-mismatch subscription.
func (decodeMismatch) Topic() string { return "decode.mismatch" }

type decodeMismatchBad struct {
	Count string `json:"count"`
}

// Topic deliberately shares the subscription topic while presenting an incompatible field type.
func (decodeMismatchBad) Topic() string { return "decode.mismatch" }

// TestPublishReturnsCodecUnmarshalError verifies decoding failures reach the publisher unchanged.
func TestPublishReturnsCodecUnmarshalError(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	_, err = bus.Subscribe(func(event decodeMismatch) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(decodeMismatchBad{Count: "bad"}); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// TestSubscriptionCloseStopsDelivery verifies closed synchronous subscriptions receive no later events.
func TestSubscriptionCloseStopsDelivery(t *testing.T) {
	bus, err := NewSync()
	if err != nil {
		t.Fatalf("NewSync returned error: %v", err)
	}
	calls := 0
	sub, err := bus.Subscribe(func(userCreated) { calls++ })
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0", calls)
	}
}
