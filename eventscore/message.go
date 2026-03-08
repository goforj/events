package eventscore

// Message is the transport-facing envelope passed to drivers.
type Message struct {
	Topic   string
	Payload []byte
}
