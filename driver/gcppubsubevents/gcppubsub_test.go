package gcppubsubevents

import (
	"context"
	"testing"

	gpubsub "cloud.google.com/go/pubsub"
	"github.com/goforj/events/eventscore"
)

func TestNewRequiresProjectID(t *testing.T) {
	if _, err := New(context.Background(), Config{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewRequiresURIWithoutClient(t *testing.T) {
	if _, err := New(context.Background(), Config{ProjectID: "test-project"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestDriverConstant(t *testing.T) {
	driver := &Driver{}
	if got := driver.Driver(); got != eventscore.DriverGCPPubSub {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverGCPPubSub)
	}
}

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

func TestReadyHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := driver.Ready(ctx); err != context.Canceled {
		t.Fatalf("Ready() error = %v, want %v", err, context.Canceled)
	}
}

func TestPublishContextHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: []byte("x")})
	if err != context.Canceled {
		t.Fatalf("PublishContext() error = %v, want %v", err, context.Canceled)
	}
}

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

func TestCloseWithoutOwnedResources(t *testing.T) {
	driver := &Driver{}
	if err := driver.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestEnsureTopicHonorsContext(t *testing.T) {
	driver := &Driver{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := driver.ensureTopic(ctx, "orders.created"); err != context.Canceled {
		t.Fatalf("ensureTopic() error = %v, want %v", err, context.Canceled)
	}
}

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
	if !canceled {
		t.Fatal("expected cancel to be called")
	}
}
