package natsevents

import (
	"context"
	"testing"
	"time"

	"github.com/goforj/events/eventscore"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// TestNewRequiresURLOrConn verifies NATS construction requires a URL or injected connection.
func TestNewRequiresURLOrConn(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
}

// TestDriverConstant verifies the NATS registry identifier remains stable.
func TestDriverConstant(t *testing.T) {
	driver := &Driver{}
	if got := driver.Driver(); got != eventscore.DriverNATS {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverNATS)
	}
}

// TestNewWithInvalidURL verifies connection failures propagate from URL-based construction.
func TestNewWithInvalidURL(t *testing.T) {
	if _, err := New(Config{URL: "nats://127.0.0.1:1"}); err == nil {
		t.Fatal("expected connection error")
	}
}

// TestNewWithConn verifies injected NATS connections bypass dialing.
func TestNewWithConn(t *testing.T) {
	url, shutdown := startTestServer(t)
	defer shutdown()

	conn, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats.Connect returned error: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	driver, err := New(Config{Conn: conn})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.conn != conn {
		t.Fatal("expected New to reuse provided connection")
	}
}

// TestReadyHonorsContext verifies canceled readiness probes return before flushing NATS.
func TestReadyHonorsContext(t *testing.T) {
	url, shutdown := startTestServer(t)
	defer shutdown()

	driver, err := New(Config{URL: url})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := driver.Ready(ctx); err != context.Canceled {
		t.Fatalf("Ready() error = %v, want %v", err, context.Canceled)
	}
}

// TestPublishAndSubscribeContext verifies NATS payloads round-trip through a subscription.
func TestPublishAndSubscribeContext(t *testing.T) {
	url, shutdown := startTestServer(t)
	defer shutdown()

	driver, err := New(Config{URL: url})
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

// TestPublishContextHonorsCanceledContext verifies canceled publishes do not reach NATS.
func TestPublishContextHonorsCanceledContext(t *testing.T) {
	url, shutdown := startTestServer(t)
	defer shutdown()

	driver, err := New(Config{URL: url})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: []byte("x")})
	if err != context.Canceled {
		t.Fatalf("PublishContext() error = %v, want %v", err, context.Canceled)
	}
}

// TestSubscriptionCloseStopsDelivery verifies unsubscribing prevents future NATS callbacks.
func TestSubscriptionCloseStopsDelivery(t *testing.T) {
	url, shutdown := startTestServer(t)
	defer shutdown()

	driver, err := New(Config{URL: url})
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

// startTestServer starts an ephemeral in-process NATS server and returns complete shutdown logic.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()

	srv, err := server.NewServer(&server.Options{
		Host: "127.0.0.1",
		Port: -1,
	})
	if err != nil {
		t.Fatalf("server.NewServer returned error: %v", err)
	}

	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		srv.Shutdown()
		t.Skip("embedded NATS server unavailable in current environment")
	}

	return srv.ClientURL(), func() {
		srv.Shutdown()
		srv.WaitForShutdown()
	}
}
