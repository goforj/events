package all

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/goforj/events/driver/gcppubsubevents"
	"github.com/goforj/events/driver/kafkaevents"
	"github.com/goforj/events/driver/natsevents"
	"github.com/goforj/events/driver/natsjetstreamevents"
	"github.com/goforj/events/driver/redisevents"
	"github.com/goforj/events/driver/snsevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

type driverFixture struct {
	name    string
	enabled bool
	factory scenario.Factory
}

func TestDriverContract_AllDrivers(t *testing.T) {
	for _, fixture := range integrationFixtures(t) {
		if !fixture.enabled {
			continue
		}
		t.Run(fixture.name, func(t *testing.T) {
			scenarioProgressf("starting integration battery for driver=%s", fixture.name)
			scenario.RunDriverContract(t, fixture.factory)
		})
	}
}

func integrationFixtures(t *testing.T) []driverFixture {
	t.Helper()

	selected := selectedDrivers()

	return []driverFixture{
		{
			name:    "gcppubsub",
			enabled: selected["gcppubsub"],
			factory: func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
				t.Helper()
				scenarioProgressf("booting gcppubsub test environment")
				env, err := testenv.StartGCPPubSub(ctx)
				if err != nil {
					t.Fatalf("StartGCPPubSub returned error: %v", err)
				}
				t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := gcppubsubevents.New(ctx, gcppubsubevents.Config{
					ProjectID: env.ProjectID,
					URI:       env.URI,
				})
				if err != nil {
					t.Fatalf("gcppubsubevents.New returned error: %v", err)
				}
				t.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "kafka",
			enabled: selected["kafka"],
			factory: func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
				t.Helper()
				scenarioProgressf("booting kafka test environment")
				env, err := testenv.StartKafka(ctx)
				if err != nil {
					t.Fatalf("StartKafka returned error: %v", err)
				}
				t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := kafkaevents.New(kafkaevents.Config{Brokers: env.Brokers})
				if err != nil {
					t.Fatalf("kafkaevents.New returned error: %v", err)
				}
				t.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "natsjetstream",
			enabled: selected["natsjetstream"],
			factory: func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
				t.Helper()
				scenarioProgressf("booting natsjetstream test environment")
				env, err := testenv.StartNATSJetStream(ctx)
				if err != nil {
					t.Fatalf("StartNATSJetStream returned error: %v", err)
				}
				t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := natsjetstreamevents.New(natsjetstreamevents.Config{URL: env.URL})
				if err != nil {
					t.Fatalf("natsjetstreamevents.New returned error: %v", err)
				}
				t.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "nats",
			enabled: selected["nats"],
			factory: func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
				t.Helper()
				scenarioProgressf("booting nats test environment")
				env, err := testenv.StartNATS(ctx)
				if err != nil {
					t.Fatalf("StartNATS returned error: %v", err)
				}
				t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := natsevents.New(natsevents.Config{URL: env.URL})
				if err != nil {
					t.Fatalf("natsevents.New returned error: %v", err)
				}
				t.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "redis",
			enabled: selected["redis"],
			factory: func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
				t.Helper()
				scenarioProgressf("booting redis test environment")
				env, err := testenv.StartRedis(ctx)
				if err != nil {
					t.Fatalf("StartRedis returned error: %v", err)
				}
				t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := redisevents.New(redisevents.Config{Addr: env.Addr})
				if err != nil {
					t.Fatalf("redisevents.New returned error: %v", err)
				}
				t.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
		{
			name:    "sns",
			enabled: selected["sns"],
			factory: func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
				t.Helper()
				scenarioProgressf("booting sns test environment")
				env, err := testenv.StartSNS(ctx)
				if err != nil {
					t.Fatalf("StartSNS returned error: %v", err)
				}
				t.Cleanup(func() { _ = env.Container.Terminate(context.Background()) })

				driver, err := snsevents.New(snsevents.Config{
					Region:   env.Region,
					Endpoint: env.Endpoint,
				})
				if err != nil {
					t.Fatalf("snsevents.New returned error: %v", err)
				}
				t.Cleanup(func() { _ = driver.Close() })
				return driver
			},
		},
	}
}

func scenarioProgressf(format string, args ...any) {
	if testing.Verbose() {
		fmt.Fprintf(os.Stderr, "[integration] "+format+"\n", args...)
	}
}

func selectedDrivers() map[string]bool {
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
