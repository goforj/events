package redisevents

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/goforj/events/eventscore"
	"github.com/redis/go-redis/v9"
)

// TestNewRequiresAddrOrClient verifies Redis construction requires an address or injected client.
func TestNewRequiresAddrOrClient(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
}

// TestDriverConstant verifies the Redis registry identifier remains stable.
func TestDriverConstant(t *testing.T) {
	driver := &Driver{}
	if got := driver.Driver(); got != eventscore.DriverRedis {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverRedis)
	}
}

// TestNewWithAddr verifies address-based construction creates an owned Redis client.
func TestNewWithAddr(t *testing.T) {
	driver, err := New(Config{Addr: "127.0.0.1:6379"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.client == nil {
		t.Fatal("expected client")
	}
	_ = driver.Close()
}

// TestNewWithClient verifies injected Redis clients bypass address validation.
func TestNewWithClient(t *testing.T) {
	srv := startMiniRedis(t)

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	driver, err := New(Config{Client: client})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.client != client {
		t.Fatal("expected New to reuse provided client")
	}
}

// TestReady verifies Redis readiness delegates to Ping with caller context.
func TestReady(t *testing.T) {
	srv := startMiniRedis(t)

	driver, err := New(Config{Addr: srv.Addr()})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	if err := driver.Ready(context.Background()); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
}

// TestPublishAndSubscribeContext verifies Redis payloads round-trip through Pub/Sub.
func TestPublishAndSubscribeContext(t *testing.T) {
	srv := startMiniRedis(t)

	driver, err := New(Config{Addr: srv.Addr()})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	received := make(chan eventscore.Message, 1)
	sub, err := driver.SubscribeContext(context.Background(), "orders.created", func(_ context.Context, msg eventscore.Message) error {
		received <- msg
		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeContext returned error: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	if err := driver.PublishContext(context.Background(), eventscore.Message{
		Topic:   "orders.created",
		Payload: []byte(`{"id":"123"}`),
	}); err != nil {
		t.Fatalf("PublishContext returned error: %v", err)
	}

	select {
	case msg := <-received:
		if msg.Topic != "orders.created" {
			t.Fatalf("Topic = %q, want %q", msg.Topic, "orders.created")
		}
		if string(msg.Payload) != `{"id":"123"}` {
			t.Fatalf("Payload = %q", string(msg.Payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for subscribed message")
	}
}

// TestSubscribeContextHonorsCanceledContext verifies canceled subscriptions do not start delivery.
func TestSubscribeContextHonorsCanceledContext(t *testing.T) {
	srv := startMiniRedis(t)

	driver, err := New(Config{Addr: srv.Addr()})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := driver.SubscribeContext(ctx, "orders.created", func(context.Context, eventscore.Message) error {
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("SubscribeContext() error = %v, want wrapping %v", err, context.Canceled)
	}
}

// TestSubscriptionCloseStopsDelivery verifies closing Redis subscriptions terminates future callbacks.
func TestSubscriptionCloseStopsDelivery(t *testing.T) {
	srv := startMiniRedis(t)

	driver, err := New(Config{Addr: srv.Addr()})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	received := make(chan struct{}, 1)
	sub, err := driver.SubscribeContext(context.Background(), "orders.created", func(_ context.Context, msg eventscore.Message) error {
		received <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeContext returned error: %v", err)
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	if err := driver.PublishContext(context.Background(), eventscore.Message{
		Topic:   "orders.created",
		Payload: []byte(`{"id":"123"}`),
	}); err != nil {
		t.Fatalf("PublishContext returned error: %v", err)
	}

	select {
	case <-received:
		t.Fatal("received message after closing subscription")
	case <-time.After(300 * time.Millisecond):
	}
}

// startMiniRedis starts an isolated in-process server and registers deterministic shutdown.
func startMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("embedded Redis server unavailable in current environment: %v", err)
	}
	t.Cleanup(srv.Close)
	return srv
}
