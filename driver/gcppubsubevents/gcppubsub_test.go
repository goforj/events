package gcppubsubevents

import (
	"context"
	"testing"

	gpubsub "cloud.google.com/go/pubsub"
	"github.com/goforj/events/eventscore"
)

// TestNewRequiresProjectID verifies Pub/Sub construction rejects an empty project identity.
func TestNewRequiresProjectID(t *testing.T) {
	if _, err := New(context.Background(), Config{}); err == nil {
		t.Fatal("expected error")
	}
}

// TestNewRequiresURIWithoutClient verifies self-managed clients require an emulator or service URI.
func TestNewRequiresURIWithoutClient(t *testing.T) {
	if _, err := New(context.Background(), Config{ProjectID: "test-project"}); err == nil {
		t.Fatal("expected error")
	}
}

// TestDriverConstant verifies the Pub/Sub registry identifier remains stable.
func TestDriverConstant(t *testing.T) {
	driver := &Driver{}
	if got := driver.Driver(); got != eventscore.DriverGCPPubSub {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverGCPPubSub)
	}
}

// TestNewWithClient verifies injected Pub/Sub clients bypass transport construction.
func TestNewWithClient(t *testing.T) {
	client := &gpubsub.Client{}
	driver, err := New(context.Background(), Config{
		ProjectID: "test-project",
		Client:    client,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.client != client {
		t.Fatal("expected New to reuse provided client")
	}
}

// TestReadyHonorsContext verifies canceled readiness probes stop before API access.
func TestReadyHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := driver.Ready(ctx); err != context.Canceled {
		t.Fatalf("Ready() error = %v, want %v", err, context.Canceled)
	}
}

// TestPublishContextHonorsContext verifies canceled publishes do not reach Pub/Sub.
func TestPublishContextHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: []byte("x")})
	if err != context.Canceled {
		t.Fatalf("PublishContext() error = %v, want %v", err, context.Canceled)
	}
}

// TestSubscribeContextHonorsContext verifies canceled subscriptions allocate no receiver resources.
func TestSubscribeContextHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := driver.SubscribeContext(ctx, "orders.created", func(context.Context, eventscore.Message) error {
		return nil
	}); err != context.Canceled {
		t.Fatalf("SubscribeContext() error = %v, want %v", err, context.Canceled)
	}
}

// TestSubscribeContextRejectsNilHandler verifies required callbacks fail before backend access.
func TestSubscribeContextRejectsNilHandler(t *testing.T) {
	if _, err := (&Driver{}).SubscribeContext(context.Background(), "orders.created", nil); err == nil {
		t.Fatal("expected nil handler error")
	}
}

// TestCloseWithoutOwnedResources verifies injected clients are not closed by the driver.
func TestCloseWithoutOwnedResources(t *testing.T) {
	driver := &Driver{}
	if err := driver.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// TestEnsureTopicHonorsContext verifies topic creation stops when its context is canceled.
func TestEnsureTopicHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := driver.ensureTopic(ctx, "orders.created"); err != context.Canceled {
		t.Fatalf("ensureTopic() error = %v, want %v", err, context.Canceled)
	}
}

// TestSubscriptionCloseWithCanceledReceive verifies receiver cancellation does not prevent deterministic cleanup.
func TestSubscriptionCloseWithCanceledReceive(t *testing.T) {
	done := make(chan error, 1)
	done <- context.Canceled
	canceled := false
	sub := subscription{
		cancel: func() { canceled = true },
		done:   done,
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if !canceled {
		t.Fatal("expected cancel to be called")
	}
}
