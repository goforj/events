package root

import (
	"testing"

	"github.com/goforj/events"
	"github.com/goforj/events/eventstest"
)

func TestSyncContractIntegration(t *testing.T) {
	eventstest.RunBusContract(t, func(testing.TB) events.API {
		bus, err := events.NewSync()
		if err != nil {
			t.Fatalf("NewSync returned error: %v", err)
		}
		return bus
	})
}
