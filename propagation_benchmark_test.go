package events

import (
	"context"
	"fmt"
	"testing"
)

// benchmarkPropagationKey keeps publisher identity separate from extracted identity so local delivery cannot hide missing extraction.
type benchmarkPropagationKey string

// benchmarkContextPropagator models an opaque two-field carrier without adding a tracing SDK dependency to the package.
type benchmarkContextPropagator struct{}

// Inject gives each publication a fresh carrier like an application propagator would produce.
func (benchmarkContextPropagator) Inject(ctx context.Context) map[string]string {
	identity, _ := ctx.Value(benchmarkPropagationKey("publisher")).(string)
	if identity == "" {
		return nil
	}
	return map[string]string{"traceparent": identity, "tracestate": "vendor=benchmark"}
}

// Extract binds only transported identity so the benchmark verifies the receiver path independently of shared context values.
func (benchmarkContextPropagator) Extract(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, benchmarkPropagationKey("receiver"), headers["traceparent"])
}

// benchmarkPropagationEvent represents a small business event rather than an empty reflection-only payload.
type benchmarkPropagationEvent struct {
	OrderID   string `json:"order_id"`
	AccountID string `json:"account_id"`
	Total     int    `json:"total"`
}

// Topic keeps routing work identical across propagation modes.
func (benchmarkPropagationEvent) Topic() string { return "orders.created" }

// BenchmarkContextPropagationPublish compares warm typed delivery with absent, empty, and populated propagation carriers.
func BenchmarkContextPropagationPublish(b *testing.B) {
	const identity = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	for _, mode := range []string{"disabled", "configured_empty", "valid_carrier"} {
		for _, subscribers := range []int{0, 1, 4} {
			b.Run(fmt.Sprintf("%s/subscribers_%d", mode, subscribers), func(b *testing.B) {
				var options []Option
				if mode != "disabled" {
					options = append(options, WithContextPropagator(benchmarkContextPropagator{}))
				}
				bus, err := NewSync(options...)
				if err != nil {
					b.Fatal(err)
				}
				ctx := context.Background()
				wantIdentity := ""
				if mode == "valid_carrier" {
					ctx = context.WithValue(ctx, benchmarkPropagationKey("publisher"), identity)
					wantIdentity = identity
				}
				handle := bus.WithContext(ctx)
				event := benchmarkPropagationEvent{OrderID: "order-42", AccountID: "account-7", Total: 12900}
				calls := 0
				var received benchmarkPropagationEvent
				var receivedIdentity string
				for range subscribers {
					subscription, err := bus.Subscribe(func(ctx context.Context, value benchmarkPropagationEvent) {
						calls++
						received = value
						receivedIdentity, _ = ctx.Value(benchmarkPropagationKey("receiver")).(string)
					})
					if err != nil {
						b.Fatal(err)
					}
					b.Cleanup(func() {
						if err := subscription.Close(); err != nil {
							b.Error(err)
						}
					})
				}
				if err := handle.Publish(event); err != nil {
					b.Fatal(err)
				}
				if calls != subscribers || subscribers > 0 && (received != event || receivedIdentity != wantIdentity) {
					b.Fatalf("warm delivery calls=%d payload=%+v carrier=%q", calls, received, receivedIdentity)
				}
				calls = 0
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if err := handle.Publish(event); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				if calls != b.N*subscribers || subscribers > 0 && (received != event || receivedIdentity != wantIdentity) {
					b.Fatalf("measured delivery calls=%d payload=%+v carrier=%q", calls, received, receivedIdentity)
				}
				b.ReportMetric(float64(subscribers), "deliveries/op")
			})
		}
	}
}
