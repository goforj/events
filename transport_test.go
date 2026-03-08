package events

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/goforj/events/eventscore"
)

type fakeTransport struct {
	mu           sync.Mutex
	published    []eventscore.Message
	subscribers  map[string]eventscore.MessageHandler
	readyErr     error
	publishErr   error
	subscribeErr error
	lastCtx      context.Context
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{subscribers: make(map[string]eventscore.MessageHandler)}
}

func (f *fakeTransport) Driver() eventscore.Driver { return eventscore.DriverNATS }
func (f *fakeTransport) Ready(context.Context) error {
	return f.readyErr
}
func (f *fakeTransport) PublishContext(ctx context.Context, msg eventscore.Message) error {
	f.lastCtx = ctx
	if f.publishErr != nil {
		return f.publishErr
	}
	f.mu.Lock()
	f.published = append(f.published, msg)
	handler := f.subscribers[msg.Topic]
	f.mu.Unlock()
	if handler != nil {
		return handler(context.Background(), msg)
	}
	return nil
}
func (f *fakeTransport) SubscribeContext(_ context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subscribers[topic] = handler
	return fakeTransportSubscription(func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.subscribers, topic)
	}), nil
}

type fakeTransportSubscription func()

func (f fakeTransportSubscription) Close() error {
	f()
	return nil
}

func TestBusWithTransportDelegatesReady(t *testing.T) {
	want := errors.New("not ready")
	transport := newFakeTransport()
	transport.readyErr = want
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := bus.Ready(); !errors.Is(err, want) {
		t.Fatalf("Ready error = %v, want %v", err, want)
	}
}

func TestBusWithTransportPublishesAndReceives(t *testing.T) {
	transport := newFakeTransport()
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	called := false
	_, err = bus.Subscribe(func(userCreated) { called = true })
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := bus.Publish(userCreated{}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if !called {
		t.Fatal("expected transport-backed delivery")
	}
	if len(transport.published) != 1 {
		t.Fatalf("published count = %d, want 1", len(transport.published))
	}
	if transport.published[0].Topic != "user.created" {
		t.Fatalf("published topic = %q, want %q", transport.published[0].Topic, "user.created")
	}
}

func TestTransportSubscriptionClosesWhenLastHandlerRemoved(t *testing.T) {
	transport := newFakeTransport()
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	sub, err := bus.Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	transport.mu.Lock()
	_, ok := transport.subscribers["user.created"]
	transport.mu.Unlock()
	if ok {
		t.Fatal("expected transport subscriber removal")
	}
}

func TestBusWithTransportPropagatesPublishError(t *testing.T) {
	want := errors.New("publish failed")
	transport := newFakeTransport()
	transport.publishErr = want

	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := bus.Publish(userCreated{}); !errors.Is(err, want) {
		t.Fatalf("Publish error = %v, want %v", err, want)
	}
}

func TestBusWithTransportUsesBackgroundContextForNilPublishContext(t *testing.T) {
	transport := newFakeTransport()
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := bus.PublishContext(nil, userCreated{}); err != nil {
		t.Fatalf("PublishContext returned error: %v", err)
	}
	if transport.lastCtx == nil {
		t.Fatal("expected non-nil context to be forwarded")
	}
}

func TestBusWithTransportRollsBackHandlerOnSubscribeError(t *testing.T) {
	want := errors.New("subscribe failed")
	transport := newFakeTransport()
	transport.subscribeErr = want

	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := bus.Subscribe(func(userCreated) {}); !errors.Is(err, want) {
		t.Fatalf("Subscribe error = %v, want %v", err, want)
	}

	bus.mu.RLock()
	defer bus.mu.RUnlock()
	if len(bus.handlers["user.created"]) != 0 {
		t.Fatal("expected failed subscribe to roll back registered handler")
	}
}
