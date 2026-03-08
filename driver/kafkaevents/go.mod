module github.com/goforj/events/driver/kafkaevents

go 1.25.0

require (
	github.com/goforj/events/eventscore v0.0.0
	github.com/segmentio/kafka-go v0.4.50
)

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
)

replace github.com/goforj/events => ../..

replace github.com/goforj/events/eventscore => ../../eventscore
