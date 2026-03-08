package events

import (
	"context"
	"errors"
	"testing"
)

func TestNewRegisteredHandlerRejectsInvalidSignatures(t *testing.T) {
	tests := []any{
		nil,
		42,
		func() {},
		func(string, string, string) {},
		func(context.Context, userCreated) string { return "" },
		func(v any) {},
	}
	for _, test := range tests {
		if _, err := newRegisteredHandler(test); err == nil {
			t.Fatalf("expected error for handler %#v", test)
		}
	}
}

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

func (c codecFailure) Marshal(any) ([]byte, error) { return nil, c.err }
func (c codecFailure) Unmarshal([]byte, any) error { return c.err }

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

func (decodeMismatch) Topic() string { return "decode.mismatch" }

type decodeMismatchBad struct {
	Count string `json:"count"`
}

func (decodeMismatchBad) Topic() string { return "decode.mismatch" }

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
