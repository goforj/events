package eventscore

// Message is the transport-facing envelope passed to drivers.
type Message struct {
	// Topic identifies the transport routing destination.
	Topic string
	// Payload contains the codec-produced event bytes without a driver envelope.
	Payload []byte
}
