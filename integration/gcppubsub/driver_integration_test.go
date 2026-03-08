package gcppubsub

import (
	"context"
	"testing"

	"github.com/goforj/events/driver/gcppubsubevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

func TestGCPPubSubDriverIntegration(t *testing.T) {
	scenario.RunDriverContract(t, func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
		t.Helper()

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
	})
}
