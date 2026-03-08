package nats

import (
	"context"
	"testing"

	"github.com/goforj/events/driver/natsevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

func TestNATSDriverIntegration(t *testing.T) {
	scenario.RunDriverContract(t, func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
		t.Helper()

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
	})
}
