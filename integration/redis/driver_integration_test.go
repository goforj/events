package redis

import (
	"context"
	"testing"

	"github.com/goforj/events/driver/redisevents"
	"github.com/goforj/events/eventscore"
	"github.com/goforj/events/integration/scenario"
	"github.com/goforj/events/integration/testenv"
)

func TestRedisDriverIntegration(t *testing.T) {
	scenario.RunDriverContract(t, func(t testing.TB, ctx context.Context) eventscore.DriverAPI {
		t.Helper()

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
	})
}
