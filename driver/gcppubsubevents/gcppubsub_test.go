package gcppubsubevents

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net"
	"strings"
	"testing"
	"time"

	gpubsub "cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	"github.com/goforj/events/eventscore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// TestPublishHeadersBoundaries verifies native attribute limits without relying on emulator enforcement.
func TestPublishHeadersBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		invalid bool
	}{
		{name: "key at limit", key: strings.Repeat("k", 256), value: "value"},
		{name: "key over limit", key: strings.Repeat("k", 257), value: "value", invalid: true},
		{name: "value at limit", key: "key", value: strings.Repeat("v", 1024)},
		{name: "value over limit", key: "key", value: strings.Repeat("v", 1025), invalid: true},
		{name: "multibyte key at limit", key: strings.Repeat("é", 128), value: "value"},
		{name: "multibyte key over limit", key: strings.Repeat("é", 128) + "k", value: "value", invalid: true},
		{name: "multibyte value at limit", key: "key", value: strings.Repeat("é", 512)},
		{name: "multibyte value over limit", key: "key", value: strings.Repeat("é", 512) + "v", invalid: true},
		{name: "reserved prefix", key: "googcustom", value: "value", invalid: true},
		{name: "reserved exact key", key: "goog", value: "value", invalid: true},
		{name: "uppercase prefix", key: "Googcustom", value: "value"},
		{name: "embedded prefix", key: "customgoog", value: "value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"traceparent": "valid", tt.key: tt.value}
			got := publishHeaders(headers)
			if tt.invalid {
				if got != nil {
					t.Fatalf("publishHeaders() = %v, want entire carrier dropped", got)
				}
			} else if !maps.Equal(got, headers) {
				t.Fatalf("publishHeaders() = %v, want %v", got, headers)
			}
			if headers[tt.key] != tt.value || headers["traceparent"] != "valid" {
				t.Fatal("publishHeaders mutated the caller's metadata")
			}
		})
	}
}

// TestPublishHeadersCount verifies direct driver metadata obeys the native attribute count limit.
func TestPublishHeadersCount(t *testing.T) {
	for _, count := range []int{100, 101} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			headers := make(map[string]string, count)
			for i := range count {
				headers[fmt.Sprintf("key%d", i)] = "value"
			}
			got := publishHeaders(headers)
			if count == 100 && !maps.Equal(got, headers) {
				t.Fatal("expected all 100 attributes to survive")
			}
			if count == 101 && got != nil {
				t.Fatal("expected all 101 attributes to be dropped")
			}
		})
	}
}

// TestPublishHeadersSnapshot verifies validated metadata never aliases caller-owned maps.
func TestPublishHeadersSnapshot(t *testing.T) {
	if got := publishHeaders(nil); got != nil {
		t.Fatalf("publishHeaders(nil) = %v, want nil", got)
	}
	headers := map[string]string{"traceparent": "original"}
	got := publishHeaders(headers)
	headers["traceparent"] = "changed"
	if got["traceparent"] != "original" {
		t.Fatal("caller mutation changed the publish snapshot")
	}
	got["traceparent"] = "snapshot change"
	if headers["traceparent"] != "changed" {
		t.Fatal("snapshot mutation changed the caller's metadata")
	}
	empty := map[string]string{}
	if got := publishHeaders(empty); got == nil || len(got) != 0 {
		t.Fatalf("publishHeaders(empty) = %v, want non-nil empty map", got)
	}
}

// publishCaptureServer captures SDK requests to verify adapter output without emulating service validation.
type publishCaptureServer struct {
	pubsubpb.UnimplementedPublisherServer
	requests chan *pubsubpb.PublishRequest
}

// Publish records the complete request before acknowledging the SDK's publication.
func (s *publishCaptureServer) Publish(_ context.Context, req *pubsubpb.PublishRequest) (*pubsubpb.PublishResponse, error) {
	s.requests <- req
	return &pubsubpb.PublishResponse{MessageIds: []string{"message-id"}}, nil
}

// TestPublishContextPreservesPayload verifies rejected optional metadata cannot alter the SDK's business payload.
func TestPublishContextPreservesPayload(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	capture := &publishCaptureServer{requests: make(chan *pubsubpb.PublishRequest, 1)}
	pubsubpb.RegisterPublisherServer(server, capture)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///pubsub-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := gpubsub.NewClient(ctx, "test-project", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	driver, err := New(ctx, Config{ProjectID: "test-project", Client: client})
	if err != nil {
		t.Fatal(err)
	}
	topic := client.Topic("orders.created")
	driver.configureTopic(topic)
	driver.topics["orders.created"] = topic
	t.Cleanup(func() { _ = driver.Close() })
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprint("invalid=", invalid), func(t *testing.T) {
			headers := map[string]string{"traceparent": "valid"}
			if invalid {
				headers["custom"] = strings.Repeat("v", 1025)
			}
			payload := []byte("{\"order_id\":42}")
			if err := driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: payload, Headers: headers}); err != nil {
				t.Fatalf("PublishContext() error = %v", err)
			}
			req := <-capture.requests
			if len(req.Messages) != 1 {
				t.Fatalf("published %d messages, want 1", len(req.Messages))
			}
			msg := req.Messages[0]
			if !bytes.Equal(msg.Data, payload) || req.Topic != "projects/test-project/topics/orders.created" {
				t.Fatalf("published payload/topic changed: %v", req)
			}
			if invalid && len(msg.Attributes) != 0 {
				t.Fatalf("invalid optional headers reached SDK request: %v", msg.Attributes)
			}
			if !invalid && !maps.Equal(msg.Attributes, headers) {
				t.Fatalf("valid headers = %v, want %v", msg.Attributes, headers)
			}
		})
	}
}

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
