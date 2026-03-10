module github.com/goforj/events/driver/natsjetstreamevents

go 1.25.0

require (
	github.com/goforj/events/eventscore v0.0.0
	github.com/nats-io/nats-server/v2 v2.12.0
	github.com/nats-io/nats.go v1.46.1
)

replace github.com/goforj/events => ../..

replace github.com/goforj/events/eventscore => ../../eventscore
