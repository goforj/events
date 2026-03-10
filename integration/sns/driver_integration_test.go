package sns

import (
	"context"
	"testing"

	"github.com/goforj/events/driver/snsevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

func TestSNSDriverIntegration(t *testing.T) {
	scenario.RunDriverContract(t, func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
		t.Helper()

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
	})
}
