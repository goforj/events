package eventstest

import (
	"testing"

	"github.com/goforj/events"
)

func TestRunBusContract(t *testing.T) {
	RunBusContract(t, func(testing.TB) events.API {
		bus, err := events.NewSync()
		if err != nil {
			t.Fatalf("NewSync returned error: %v", err)
		}
		return bus
	})
}

func TestRunNullBusContract(t *testing.T) {
	RunNullBusContract(t, func(testing.TB) events.API {
		bus, err := events.NewNull()
		if err != nil {
			t.Fatalf("NewNull returned error: %v", err)
		}
		return bus
	})
}
