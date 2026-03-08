module github.com/goforj/events/eventsfake

go 1.24.0

require (
	github.com/goforj/events v0.0.0
	github.com/goforj/events/eventscore v0.0.0
	github.com/goforj/events/eventstest v0.0.0
)

replace github.com/goforj/events => ..

replace github.com/goforj/events/eventstest => ../eventstest

replace github.com/goforj/events/eventscore => ../eventscore
