package events

import "github.com/goforj/events/eventscore"

// Config configures root bus construction.
// @group Construction
//
// Example: define bus construction config
//
//	cfg := events.Config{Driver: eventscore.DriverSync}
//	fmt.Println(cfg.Driver)
//	// Output: sync
type Config struct {
	// Driver selects the root bus backend.
	Driver eventscore.Driver
	// Codec overrides the default JSON codec.
	Codec Codec
	// Transport installs a driver-backed transport for distributed delivery.
	Transport eventscore.DriverAPI
}
