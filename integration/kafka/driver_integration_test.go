package kafka

import (
	"context"
	"testing"

	"github.com/goforj/events/driver/kafkaevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

func TestKafkaDriverIntegration(t *testing.T) {
	scenario.RunDriverContract(t, func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
		t.Helper()

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
	})
}
