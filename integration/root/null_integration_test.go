package root

import (
	"testing"

	"github.com/goforj/events"
	"github.com/goforj/events/eventstest"
)

// TestNullIntegrationPublishIsNoop verifies the root null bus never delivers published events.
func TestNullIntegrationPublishIsNoop(t *testing.T) {
	eventstest.RunNullBusContract(t, func(testing.TB) events.API {
		bus, err := events.NewNull()
		if err != nil {
			t.Fatalf("NewNull returned error: %v", err)
		}
		return bus
	})
}
