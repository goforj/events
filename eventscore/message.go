package eventscore

// Message is the transport-facing envelope passed to drivers.
type Message struct {
	// Headers carries optional transport metadata independently from application payload bytes.
	Headers map[string]string
	// Topic identifies the transport routing destination.
	Topic string
	// Payload contains the codec-produced event bytes without a driver envelope.
	Payload []byte
}
