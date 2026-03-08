package events

import "errors"

var (
	// ErrInvalidHandler indicates a subscribe handler has an unsupported shape.
	ErrInvalidHandler = errors.New("events: invalid handler")
	// ErrNilEvent indicates a publish call received a nil event.
	ErrNilEvent = errors.New("events: nil event")
	// ErrEmptyTopic indicates topic resolution produced an empty topic.
	ErrEmptyTopic = errors.New("events: empty topic")
)
