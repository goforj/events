package redisevents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/goforj/events/eventscore"
)

// messagePrefix reserves version-one metadata framing only for explicitly opted-in channels.
var messagePrefix = []byte{0, 'g', 'o', 'f', 'o', 'r', 'j', '-', 'e', 'v', 'e', 'n', 't', 1}

// redisMessage carries optional headers on a transport that has no native metadata channel.
type redisMessage struct {
	Headers map[string]string `json:"headers"`
	Payload []byte            `json:"payload"`
}

// encodeMessage preserves raw transport by default and escapes reserved payloads when framing is enabled.
func encodeMessage(message eventscore.Message, enabled bool) ([]byte, error) {
	if !enabled {
		if len(message.Headers) > 0 {
			return nil, fmt.Errorf("redisevents: message headers require Config.EnableHeaders")
		}
		return message.Payload, nil
	}
	if len(message.Headers) == 0 && !bytes.HasPrefix(message.Payload, messagePrefix) {
		return message.Payload, nil
	}
	// String fields and bytes cannot contain unsupported JSON values.
	encoded, _ := json.Marshal(redisMessage{Headers: message.Headers, Payload: message.Payload})
	// Cap the prefix slice so append allocates an independent frame using checked runtime growth.
	return append(messagePrefix[:len(messagePrefix):len(messagePrefix)], encoded...), nil
}

// decodeMessage accepts metadata only on opted-in channels, leaving every legacy binary payload intact by default.
func decodeMessage(payload []byte, enabled bool) (eventscore.Message, error) {
	if !enabled || !bytes.HasPrefix(payload, messagePrefix) {
		return eventscore.Message{Payload: payload}, nil
	}
	var frame *struct {
		Headers json.RawMessage `json:"headers"`
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(payload[len(messagePrefix):], &frame) != nil || frame == nil || frame.Payload == nil {
		return eventscore.Message{}, fmt.Errorf("redisevents: invalid metadata envelope")
	}
	var body []byte
	if json.Unmarshal(frame.Payload, &body) != nil {
		return eventscore.Message{}, fmt.Errorf("redisevents: invalid metadata envelope")
	}
	var headers map[string]string
	// Bound metadata before allocating its map; unusable correlation must not discard a valid business payload.
	if len(frame.Headers) > 8192 || !utf8.Valid(frame.Headers) || json.Unmarshal(frame.Headers, &headers) != nil {
		headers = nil
	}
	return eventscore.Message{Headers: headers, Payload: body}, nil
}
