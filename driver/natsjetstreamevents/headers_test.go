package natsjetstreamevents

import (
	"bytes"
	"context"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/goforj/events/eventscore"
	"github.com/nats-io/nats.go"
)

// TestCarrierHeaders exercises native fields, repeated values, and the ordinary header-free path.
func TestCarrierHeaders(t *testing.T) {
	if publishHeaders(nil) != nil || receivedHeaders(nil) != nil {
		t.Fatal("empty metadata changed the ordinary path")
	}
	headers := receivedHeaders(publishHeaders(map[string]string{"traceparent": "producer", "tracestate": "vendor=value"}))
	if headers["traceparent"] != "producer" || headers["tracestate"] != "vendor=value" {
		t.Fatal(headers)
	}
	repeated := receivedHeaders(nats.Header{"traceparent": []string{"first", "second"}})
	if repeated["traceparent"] != "first,second" {
		t.Fatal("ambiguous parents were hidden")
	}
}

// TestCarrierExcludesExpectedStream keeps driver-owned routing headers from consuming the application carrier budget.
func TestCarrierExcludesExpectedStream(t *testing.T) {
	headers := publishHeaders(map[string]string{"traceparent": "producer"})
	headers.Set(nats.ExpectedStreamHdr, "events-stream")
	actual := receivedHeaders(headers)
	if len(actual) != 1 || actual["traceparent"] != "producer" {
		t.Fatalf("application headers=%v", actual)
	}
	if actual := receivedHeaders(nats.Header{nats.ExpectedStreamHdr: []string{"events-stream"}}); len(actual) != 0 {
		t.Fatal("routing assertion became application metadata")
	}
}

// TestValidHeaderKey checks the complete byte range against the client's supported header alphabet.
func TestValidHeaderKey(t *testing.T) {
	const alphabet = "!#$%&'*+-.0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ^_`abcdefghijklmnopqrstuvwxyz|~"
	for value := 0; value <= 255; value++ {
		key := string([]byte{byte(value)})
		want := strings.Contains(alphabet, key)
		if got := validHeaderKey(key); got != want {
			t.Errorf("validHeaderKey(%q) = %v, want %v", key, got, want)
		}
	}
	for _, key := range []string{"", "trace parent", "trace\tparent", "trace\rparent", "trace\nparent", "tracé", "trace😀", "trace\x00parent", "trace\x7fparent"} {
		if validHeaderKey(key) {
			t.Errorf("validHeaderKey(%q) accepted invalid key", key)
		}
	}
	if !validHeaderKey(alphabet) {
		t.Fatal("validHeaderKey rejected combined valid characters")
	}
}

// TestPublishHeadersOmitsInvalidCarrier ensures one unusable optional key discards the whole carrier.
func TestPublishHeadersOmitsInvalidCarrier(t *testing.T) {
	if got := publishHeaders(map[string]string{}); got != nil {
		t.Fatalf("empty carrier = %v, want nil", got)
	}
	for _, key := range []string{"", " ", "trace parent", "trace:parent", "trace\r\nparent", "tracé", "trace\x00parent", "trace\x7fparent"} {
		headers := map[string]string{"traceparent": "valid", key: "invalid"}
		original := maps.Clone(headers)
		if got := publishHeaders(headers); got != nil {
			t.Errorf("publishHeaders with invalid key %q = %v, want nil", key, got)
		}
		if !maps.Equal(headers, original) {
			t.Errorf("publishHeaders changed caller headers for key %q", key)
		}
	}
}

// TestPublishHeadersPreservesValuesAndOwnership leaves value sanitation to the client and isolates caller storage.
func TestPublishHeadersPreservesValuesAndOwnership(t *testing.T) {
	const value = "  vendor=é\r\nnext  "
	headers := map[string]string{"tracestate": value, "empty": ""}
	native := publishHeaders(headers)
	if len(native) != 2 || len(native["tracestate"]) != 1 || native["tracestate"][0] != value || len(native["empty"]) != 1 || native["empty"][0] != "" {
		t.Fatalf("native headers = %v", native)
	}
	headers["tracestate"] = "caller change"
	if native["tracestate"][0] != value {
		t.Fatal("caller mutation changed native headers")
	}
	native["tracestate"][0] = "native change"
	if headers["tracestate"] != "caller change" {
		t.Fatal("native mutation changed caller headers")
	}
}

// TestPublishContextPreservesPayloadWithOptionalHeaders checks delivery when carrier keys cannot be encoded by NATS.
func TestPublishContextPreservesPayloadWithOptionalHeaders(t *testing.T) {
	url, shutdown := startJetStreamServer(t)
	defer shutdown()
	driver, err := New(Config{URL: url})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })
	received := make(chan eventscore.Message, 1)
	sub, err := driver.SubscribeContext(context.Background(), "orders.optional", func(_ context.Context, msg eventscore.Message) error {
		received <- msg
		return nil
	})
	if err != nil {
		t.Fatalf("SubscribeContext returned error: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	for _, test := range []struct {
		name    string
		headers map[string]string
		want    map[string]string
	}{
		{name: "empty key", headers: map[string]string{"": "optional", "traceparent": "valid"}},
		{name: "space key", headers: map[string]string{"trace parent": "optional", "traceparent": "valid"}},
		{name: "delimiter key", headers: map[string]string{"trace:parent": "optional", "traceparent": "valid"}},
		{name: "control key", headers: map[string]string{"trace\x00parent": "optional", "traceparent": "valid"}},
		{name: "UTF8 key", headers: map[string]string{"tracé": "optional", "traceparent": "valid"}},
		{
			name:    "client value sanitation",
			headers: map[string]string{"tracestate": "  vendor=é\r\nnext  "},
			want:    map[string]string{"tracestate": "vendor=é  next"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := []byte("payload: " + test.name + "\x00\xff")
			original := maps.Clone(test.headers)
			if err := driver.PublishContext(context.Background(), eventscore.Message{Topic: "orders.optional", Payload: payload, Headers: test.headers}); err != nil {
				t.Fatalf("PublishContext returned error: %v", err)
			}
			if !maps.Equal(test.headers, original) {
				t.Fatal("publish changed caller headers")
			}
			select {
			case msg := <-received:
				if msg.Topic != "orders.optional" || !bytes.Equal(msg.Payload, payload) {
					t.Fatalf("received topic %q payload %q, want %q payload %q", msg.Topic, msg.Payload, "orders.optional", payload)
				}
				if !maps.Equal(msg.Headers, test.want) {
					t.Fatalf("received headers = %v, want %v", msg.Headers, test.want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("timed out waiting for subscribed message")
			}
		})
	}
}
