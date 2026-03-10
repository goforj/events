package eventscore

// Driver identifies the active events backend.
type Driver string

const (
	// DriverSync dispatches in-process synchronously.
	DriverSync Driver = "sync"
	// DriverNull drops events without delivery.
	DriverNull Driver = "null"
	// DriverNATS identifies a NATS-backed transport.
	DriverNATS Driver = "nats"
	// DriverRedis identifies a Redis-backed transport.
	DriverRedis Driver = "redis"
	// DriverKafka identifies a Kafka-backed transport.
	DriverKafka Driver = "kafka"
	// DriverSNS identifies an SNS-backed transport.
	DriverSNS Driver = "sns"
	// DriverGCPPubSub identifies a Google Pub/Sub-backed transport.
	DriverGCPPubSub Driver = "gcppubsub"
	// DriverSQS identifies an SQS-backed transport.
	DriverSQS Driver = "sqs"
)
