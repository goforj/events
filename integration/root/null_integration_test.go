package root

import (
	"testing"

	"github.com/goforj/events"
	"github.com/goforj/events/eventstest"
)

func TestNullIntegrationPublishIsNoop(t *testing.T) {
	eventstest.RunNullBusContract(t, func(testing.TB) events.API {
		bus, err := events.NewNull()
		if err != nil {
			t.Fatalf("NewNull returned error: %v", err)
		}
		return bus
	})
}
