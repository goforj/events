package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// propagationKey distinguishes selected correlation from receiver-owned values.
type propagationKey string

// testPropagator snapshots one header and records extraction without a tracing dependency.
type testPropagator struct{ headers map[string]string }

// Inject returns explicit test metadata, leaving unrelated context values private.
func (p testPropagator) Inject(context.Context) map[string]string { return p.headers }

// Extract restores the selected value while preserving the caller's lifetime.
func (testPropagator) Extract(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, propagationKey("trace"), headers["traceparent"])
}

// TestContextPropagatorPreservesLocalHandlerContext verifies typed delivery and the receiver cancellation contract.
func TestContextPropagatorPreservesLocalHandlerContext(t *testing.T) {
	bus, err := NewSync(WithContextPropagator(testPropagator{headers: map[string]string{"traceparent": "producer"}}))
	if err != nil {
		t.Fatal(err)
	}
	type event struct{ ID string }
	deadline := time.Now().Add(time.Hour)
	ctx, cancel := context.WithDeadline(context.WithValue(context.Background(), propagationKey("receiver"), "retained"), deadline)
	defer cancel()
	calls := 0
	sub, err := bus.Subscribe(func(received context.Context, value event) error {
		calls++
		if received.Value(propagationKey("trace")) != "producer" || received.Value(propagationKey("receiver")) != "retained" || value.ID != "exact" {
			t.Error("handler context or payload changed")
		}
		if got, ok := received.Deadline(); !ok || !got.Equal(deadline) {
			t.Error("receiver deadline changed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	if err := bus.WithContext(ctx).Publish(event{ID: "exact"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("handler calls=%d", calls)
	}
}

// TestContextPropagatorSnapshotsTransportHeaders verifies that a real transport boundary, not shared Go context, restores correlation.
func TestContextPropagatorSnapshotsTransportHeaders(t *testing.T) {
	headers := map[string]string{"traceparent": "producer"}
	transport := newFakeTransport()
	bus, err := New(Config{Transport: transport}, WithContextPropagator(testPropagator{headers: headers}))
	if err != nil {
		t.Fatal(err)
	}
	type event struct{ ID string }
	calls := 0
	sub, err := bus.Subscribe(func(ctx context.Context, value event) error {
		calls++
		if ctx.Value(propagationKey("trace")) != "producer" || ctx.Value(propagationKey("private")) != nil || value.ID != "exact" {
			t.Error("transport lost correlation or exposed publisher context values")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()
	publisher := context.WithValue(context.Background(), propagationKey("private"), "never serialize")
	if err := bus.WithContext(publisher).Publish(event{ID: "exact"}); err != nil {
		t.Fatal(err)
	}
	headers["traceparent"] = "changed"
	if calls != 1 || transport.published[0].Headers["traceparent"] != "producer" {
		t.Fatal("transport retained mutable carrier fields")
	}
}

// TestContextCarrierLimitsPreserveBusinessDelivery checks both sides of the transport boundary for optional metadata failures.
func TestContextCarrierLimitsPreserveBusinessDelivery(t *testing.T) {
	tooMany := map[string]string{"traceparent": "producer"}
	for index := range 10 {
		tooMany[fmt.Sprintf("field%d", index)] = "value"
	}
	for _, test := range []struct {
		name    string
		headers map[string]string
	}{
		{"absent", nil},
		{"empty", map[string]string{}},
		{"too_many", tooMany},
		{"raw_size", map[string]string{"traceparent": strings.Repeat("x", 8192)}},
		{"encoded_size", map[string]string{"traceparent": strings.Repeat("x", 8180)}},
		{"escaped_size", map[string]string{"traceparent": strings.Repeat("\x00", 1500)}},
		{"invalid_key_utf8", map[string]string{"traceparent": "producer", string([]byte{255}): "value"}},
		{"invalid_value_utf8", map[string]string{"traceparent": string([]byte{255})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			transport := newFakeTransport()
			bus, err := New(Config{Transport: transport}, WithContextPropagator(testPropagator{headers: test.headers}))
			if err != nil {
				t.Fatal(err)
			}
			type event struct{ ID string }
			calls := 0
			sub, err := bus.Subscribe(func(ctx context.Context, value event) error {
				calls++
				if value.ID != "exact" || ctx.Value(propagationKey("trace")) != nil {
					t.Error("invalid optional metadata changed delivery")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Close()
			if err := bus.Publish(event{ID: "exact"}); err != nil {
				t.Fatal(err)
			}
			message := transport.published[0]
			if len(message.Headers) != 0 {
				t.Fatal("invalid metadata reached the publisher")
			}
			message.Headers = test.headers
			if err := transport.subscribers[message.Topic](context.Background(), message); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("business deliveries = %d, want 2", calls)
			}
		})
	}
}

// TestContextCarrierMaximumsRemainUsable verifies both inclusive limits and the snapshot contract.
func TestContextCarrierMaximumsRemainUsable(t *testing.T) {
	maximumFields := map[string]string{"traceparent": "producer"}
	for index := range 9 {
		maximumFields[fmt.Sprintf("field%d", index)] = "value"
	}
	maximumBytes := map[string]string{"traceparent": strings.Repeat("x", 8192-len(`{"traceparent":""}`))}
	for _, headers := range []map[string]string{maximumFields, maximumBytes} {
		copied := boundedContextCarrier(headers)
		if len(copied) != len(headers) || copied["traceparent"] != headers["traceparent"] {
			t.Fatal("valid maximum carrier was rejected")
		}
		original := copied["traceparent"]
		headers["traceparent"] = "changed"
		if copied["traceparent"] != original {
			t.Fatal("carrier retained a mutable input map")
		}
	}
}

// mutatingPropagator checks that each extraction owns its metadata snapshot.
type mutatingPropagator struct{}

// Inject supplies one field so handler isolation crosses a transport boundary.
func (mutatingPropagator) Inject(context.Context) map[string]string {
	return map[string]string{"traceparent": "producer"}
}

// Extract deliberately changes the supplied map to expose sharing between handlers.
func (mutatingPropagator) Extract(ctx context.Context, headers map[string]string) context.Context {
	value := headers["traceparent"]
	headers["traceparent"] = "modified"
	return context.WithValue(ctx, propagationKey("trace"), value)
}

// TestContextCarrierIsolatesExtractors prevents an earlier handler from changing later handlers or driver metadata.
func TestContextCarrierIsolatesExtractors(t *testing.T) {
	transport := newFakeTransport()
	bus, err := New(Config{Transport: transport}, WithContextPropagator(mutatingPropagator{}))
	if err != nil {
		t.Fatal(err)
	}
	type event struct{ ID string }
	calls := 0
	for range 2 {
		sub, err := bus.Subscribe(func(ctx context.Context, value event) error {
			calls++
			if ctx.Value(propagationKey("trace")) != "producer" || value.ID != "exact" {
				t.Error("earlier extraction changed the handler metadata")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Close()
	}
	if err := bus.Publish(event{ID: "exact"}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || transport.published[0].Headers["traceparent"] != "producer" {
		t.Fatal("handler isolation or original transport metadata changed")
	}
}

// TestContextCarrierSizeFastPathMatchesJSON checks worst-case escaping and the exact-size fallback against the encoder.
func TestContextCarrierSizeFastPathMatchesJSON(t *testing.T) {
	for _, unit := range []string{"a", "\x00", "\n", "\"", "\\", "<", ">", "&", "é", "\u2028", "\u2029", "\xff"} {
		for _, length := range []int{0, 1, 225, 226, 1350, 1360, 1361, 1362, 1363, 1364, 4090, 8170, 8192} {
			for _, key := range []string{"traceparent", unit} {
				headers := map[string]string{key: strings.Repeat(unit, length)}
				encoded, err := json.Marshal(headers)
				if err != nil {
					t.Fatal(err)
				}
				want := utf8.ValidString(key) && utf8.ValidString(headers[key]) && len(key)+len(headers[key]) <= 8192 && len(encoded) <= 8192
				if got := validContextCarrier(headers); got != want {
					t.Fatalf("unit=%q length=%d key=%q: valid=%v want=%v encoded=%d", unit, length, key, got, want, len(encoded))
				}
			}
		}
	}
}
