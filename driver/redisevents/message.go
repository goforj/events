package redisevents

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/goforj/events/eventscore"
)

// messagePrefix reserves version-one metadata framing while keeping ordinary Redis publications byte-for-byte compatible.
var messagePrefix = []byte{0, 'g', 'o', 'f', 'o', 'r', 'j', '-', 'e', 'v', 'e', 'n', 't', 1}

// redisMessage carries optional headers on a transport that has no native metadata channel.
type redisMessage struct {
	Headers map[string]string `json:"headers"`
	Payload []byte            `json:"payload"`
}

// encodeMessage frames only header-bearing messages, preserving exact application payload bytes inside that frame.
func encodeMessage(message eventscore.Message) []byte {
	if len(message.Headers) == 0 {
		return message.Payload
	}
	// String fields and bytes cannot contain unsupported JSON values.
	encoded, _ := json.Marshal(redisMessage{Headers: message.Headers, Payload: message.Payload})
	return append(append(make([]byte, 0, len(messagePrefix)+len(encoded)), messagePrefix...), encoded...)
}

// decodeMessage accepts legacy raw publications and the optional versioned metadata envelope.
func decodeMessage(payload []byte) (eventscore.Message, error) {
	if !bytes.HasPrefix(payload, messagePrefix) {
		return eventscore.Message{Payload: payload}, nil
	}
	var message redisMessage
	if err := json.Unmarshal(payload[len(messagePrefix):], &message); err != nil {
		return eventscore.Message{}, fmt.Errorf("redisevents: invalid metadata envelope")
	}
	return eventscore.Message{Headers: message.Headers, Payload: message.Payload}, nil
}
