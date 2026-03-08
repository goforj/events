package kafkaevents

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/goforj/events/eventscore"
	"github.com/segmentio/kafka-go"
)

func TestNewRequiresBrokers(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestDriverConstant(t *testing.T) {
	driver := &Driver{}
	if got := driver.Driver(); got != eventscore.DriverKafka {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverKafka)
	}
}

func TestNewWithWriter(t *testing.T) {
	writer := &kafka.Writer{}
	driver, err := New(Config{
		Brokers: []string{"127.0.0.1:9092"},
		Writer:  writer,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.writer != writer {
		t.Fatal("expected New to reuse provided writer")
	}
}

func TestNewDefaultsAndCopiesBrokers(t *testing.T) {
	cfg := Config{Brokers: []string{"127.0.0.1:9092"}}

	driver, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.dialer == nil {
		t.Fatal("expected New to create a default dialer")
	}
	if driver.writer == nil {
		t.Fatal("expected New to create a default writer")
	}
	if driver.writer.BatchSize != 1 {
		t.Fatalf("writer BatchSize = %d, want 1", driver.writer.BatchSize)
	}
	if driver.writer.BatchTimeout != defaultWriterBatchTimeout {
		t.Fatalf("writer BatchTimeout = %v, want %v", driver.writer.BatchTimeout, defaultWriterBatchTimeout)
	}

	cfg.Brokers[0] = "mutated:9092"
	if got := driver.brokers[0]; got != "127.0.0.1:9092" {
		t.Fatalf("driver broker = %q, want original value", got)
	}
}

func TestReadyHonorsContext(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:9092"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := driver.Ready(ctx); err != context.Canceled {
		t.Fatalf("Ready() error = %v, want %v", err, context.Canceled)
	}
}

func TestReadyReturnsDialError(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := driver.Ready(ctx); err == nil {
		t.Fatal("expected Ready to return a dial error")
	}
}

func TestPublishContextHonorsContext(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:9092"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: []byte("x")})
	if err != context.Canceled {
		t.Fatalf("PublishContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestPublishContextReturnsEnsureTopicError(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: []byte("x")})
	if err == nil {
		t.Fatal("expected PublishContext to return an error")
	}
}

func TestSubscribeContextHonorsContext(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:9092"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := driver.SubscribeContext(ctx, "orders.created", func(context.Context, eventscore.Message) error {
		return nil
	}); err != context.Canceled {
		t.Fatalf("SubscribeContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestSubscribeContextReturnsEnsureTopicError(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = driver.SubscribeContext(ctx, "orders.created", func(context.Context, eventscore.Message) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected SubscribeContext to return an error")
	}
}

func TestEnsureTopicHonorsContext(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:9092"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := driver.ensureTopic(ctx, "orders.created"); err != context.Canceled {
		t.Fatalf("ensureTopic() error = %v, want %v", err, context.Canceled)
	}
}

func TestEnsureTopicReturnsDialError(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = driver.ensureTopic(ctx, "orders.created")
	if err == nil {
		t.Fatal("expected ensureTopic to return an error")
	}
	if strings.TrimSpace(err.Error()) == "" {
		t.Fatal("expected ensureTopic error to have a message")
	}
}

func TestEnsureTopicSkipsDialWhenAlreadyCached(t *testing.T) {
	driver, err := New(Config{Brokers: []string{"127.0.0.1:1"}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	driver.ensuredTopics["orders.created"] = struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := driver.ensureTopic(ctx, "orders.created"); err != nil {
		t.Fatalf("ensureTopic() error = %v, want nil for cached topic", err)
	}
}

func TestClose(t *testing.T) {
	driver, err := New(Config{
		Brokers: []string{"127.0.0.1:9092"},
		Writer:  &kafka.Writer{},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := driver.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestSubscriptionClose(t *testing.T) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"127.0.0.1:9092"},
		Topic:     "orders.created",
		Partition: 0,
	})
	done := make(chan struct{})
	close(done)

	ctx, cancel := context.WithCancel(context.Background())
	sub := subscription{
		cancel: cancel,
		done:   done,
		reader: reader,
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if ctx.Err() != context.Canceled {
		t.Fatalf("context error = %v, want %v", ctx.Err(), context.Canceled)
	}
}
