package events

import "errors"

var (
	// ErrInvalidHandler indicates a subscribe handler has an unsupported shape.
	ErrInvalidHandler = errors.New("events: invalid handler")
	// ErrNilEvent indicates a publish call received a nil event.
	ErrNilEvent = errors.New("events: nil event")
	// ErrEmptyTopic indicates topic resolution produced an empty topic.
	ErrEmptyTopic = errors.New("events: empty topic")
	// ErrUnsupportedDriver indicates a local bus was configured with a transport-only driver.
	ErrUnsupportedDriver = errors.New("events: unsupported local driver")
	// ErrNilSubscription indicates a transport accepted registration without returning a subscription.
	ErrNilSubscription = errors.New("events: transport returned nil subscription")
	// ErrNilTransport indicates a typed-nil transport was passed as a required collaborator.
	ErrNilTransport = errors.New("events: transport must not be nil")
	// ErrNilCodec indicates a typed-nil codec was passed as a required collaborator.
	ErrNilCodec = errors.New("events: codec must not be nil")
)
