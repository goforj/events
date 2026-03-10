package natsjetstream

import (
	"context"
	"testing"

	"github.com/goforj/events/driver/natsjetstreamevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

func TestNATSJetStreamDriverIntegration(t *testing.T) {
	scenario.RunDriverContract(t, func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
		t.Helper()

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
	})
}
