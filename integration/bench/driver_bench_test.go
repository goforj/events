package bench

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/goforj/events"
	"github.com/goforj/events/driver/gcppubsubevents"
	"github.com/goforj/events/driver/kafkaevents"
	"github.com/goforj/events/driver/natsevents"
	"github.com/goforj/events/driver/natsjetstreamevents"
	"github.com/goforj/events/driver/redisevents"
	"github.com/goforj/events/driver/snsevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/testenv"
)

type benchFixture struct {
	name    string
	enabled bool
	factory func(testing.TB, context.Context) eventscore.DriverAPI
}

type benchEvent struct {
	ID string `json:"id"`
}

func BenchmarkDistributedPublishRoundTrip(b *testing.B) {
	for _, fixture := range benchmarkFixtures(b) {
		if !fixture.enabled {
			continue
		}
		b.Run(fixture.name, func(b *testing.B) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			driver := fixture.factory(b, ctx)
			bus, err := events.New(events.Config{Transport: driver})
			if err != nil {
				b.Fatalf("events.New returned error: %v", err)
			}

			delivered := make(chan benchEvent, 1)
			sub, err := bus.Subscribe(func(event benchEvent) {
				select {
				case delivered <- event:
				default:
				}
			})
			if err != nil {
				b.Fatalf("Subscribe returned error: %v", err)
			}
			b.Cleanup(func() { _ = sub.Close() })

			if err := publishAndAwait(ctx, bus, delivered, "warmup"); err != nil {
				b.Fatalf("warmup round-trip failed: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := publishAndAwait(ctx, bus, delivered, "bench"); err != nil {
					b.Fatalf("timed round-trip failed: %v", err)
				}
			}
		})
	}
}

func publishAndAwait(ctx context.Context, bus events.API, delivered <-chan benchEvent, id string) error {
	payload := benchEvent{ID: id}
	if err := bus.PublishContext(ctx, payload); err != nil {
		return err
	}
	select {
	case event := <-delivered:
		if event.ID != id {
			return fmt.Errorf("event ID = %q, want %q", event.ID, id)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timed out waiting for distributed delivery: %w", ctx.Err())
	}
}

func benchmarkFixtures(tb testing.TB) []benchFixture {
	tb.Helper()

	selected := selectedBenchDrivers()

	return []benchFixture{
		{
			name:    "gcppubsub",
			enabled: selected["gcppubsub"],
			factory: func(tb testing.TB, ctx context.Context) eventscore.DriverAPI {
				tb.Helper()
				env, err := testenv.StartGCPPubSub(ctx)
				if err != nil {
					tb.Fatalf("StartGCPPubSub returned error: %v", err)
				}
				tb.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := gcppubsubevents.New(ctx, gcppubsubevents.Config{
					ProjectID: env.ProjectID,
					URI:       env.URI,
				})
				if err != nil {
					tb.Fatalf("gcppubsubevents.New returned error: %v", err)
				}
				tb.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "kafka",
			enabled: selected["kafka"],
			factory: func(tb testing.TB, ctx context.Context) eventscore.DriverAPI {
				tb.Helper()
				env, err := testenv.StartKafka(ctx)
				if err != nil {
					tb.Fatalf("StartKafka returned error: %v", err)
				}
				tb.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := kafkaevents.New(kafkaevents.Config{Brokers: env.Brokers})
				if err != nil {
					tb.Fatalf("kafkaevents.New returned error: %v", err)
				}
				tb.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "natsjetstream",
			enabled: selected["natsjetstream"],
			factory: func(tb testing.TB, ctx context.Context) eventscore.DriverAPI {
				tb.Helper()
				env, err := testenv.StartNATSJetStream(ctx)
				if err != nil {
					tb.Fatalf("StartNATSJetStream returned error: %v", err)
				}
				tb.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := natsjetstreamevents.New(natsjetstreamevents.Config{URL: env.URL})
				if err != nil {
					tb.Fatalf("natsjetstreamevents.New returned error: %v", err)
				}
				tb.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "nats",
			enabled: selected["nats"],
			factory: func(tb testing.TB, ctx context.Context) eventscore.DriverAPI {
				tb.Helper()
				env, err := testenv.StartNATS(ctx)
				if err != nil {
					tb.Fatalf("StartNATS returned error: %v", err)
				}
				tb.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := natsevents.New(natsevents.Config{URL: env.URL})
				if err != nil {
					tb.Fatalf("natsevents.New returned error: %v", err)
				}
				tb.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "redis",
			enabled: selected["redis"],
			factory: func(tb testing.TB, ctx context.Context) eventscore.DriverAPI {
				tb.Helper()
				env, err := testenv.StartRedis(ctx)
				if err != nil {
					tb.Fatalf("StartRedis returned error: %v", err)
				}
				tb.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := redisevents.New(redisevents.Config{Addr: env.Addr})
				if err != nil {
					tb.Fatalf("redisevents.New returned error: %v", err)
				}
				tb.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "sns",
			enabled: selected["sns"],
			factory: func(tb testing.TB, ctx context.Context) eventscore.DriverAPI {
				tb.Helper()
				env, err := testenv.StartSNS(ctx)
				if err != nil {
					tb.Fatalf("StartSNS returned error: %v", err)
				}
				tb.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := snsevents.New(snsevents.Config{
					Region:   env.Region,
					Endpoint: env.Endpoint,
				})
				if err != nil {
					tb.Fatalf("snsevents.New returned error: %v", err)
				}
				tb.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
	}
}

func selectedBenchDrivers() map[string]bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("INTEGRATION_DRIVER")))
	if value == "" {
		return map[string]bool{
			"gcppubsub":     true,
			"kafka":         true,
			"natsjetstream": true,
			"nats":          true,
			"redis":         true,
			"sns":           true,
		}
	}

	selected := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			selected[part] = true
		}
	}
	return selected
}
