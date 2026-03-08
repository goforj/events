package events

import "github.com/goforj/events/eventscore"

// Config configures root bus construction.
type Config struct {
	// Driver selects the root bus backend.
	Driver eventscore.Driver
	// Codec overrides the default JSON codec.
	Codec Codec
	// Transport installs a driver-backed transport for distributed delivery.
	Transport eventscore.DriverAPI
}
