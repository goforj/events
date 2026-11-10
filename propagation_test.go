package events

import (
	"context"
	"testing"
	"time"
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
