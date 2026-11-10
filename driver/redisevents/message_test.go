package redisevents

import (
	"bytes"
	"github.com/goforj/events/eventscore"
	"testing"
)

// TestMessageFramingPreservesLegacyAndMetadata checks that only explicit headers alter the Redis wire representation.
func TestMessageFramingPreservesLegacyAndMetadata(t *testing.T) {
	payload := []byte{0, 1, 2, 255, '{', '}'}
	raw := encodeMessage(eventscore.Message{Payload: payload})
	if !bytes.Equal(raw, payload) {
		t.Fatal("legacy publication changed")
	}
	legacy, err := decodeMessage(raw)
	if err != nil || !bytes.Equal(legacy.Payload, payload) || len(legacy.Headers) != 0 {
		t.Fatal("legacy delivery changed")
	}
	framed := encodeMessage(eventscore.Message{Payload: payload, Headers: map[string]string{"traceparent": "producer", "tracestate": "vendor=value"}})
	decoded, err := decodeMessage(framed)
	if err != nil || !bytes.Equal(decoded.Payload, payload) || decoded.Headers["traceparent"] != "producer" || decoded.Headers["tracestate"] != "vendor=value" {
		t.Fatalf("metadata round trip failed: %+v %v", decoded, err)
	}
	if _, err := decodeMessage(append(append([]byte(nil), messagePrefix...), []byte("invalid")...)); err == nil {
		t.Fatal("malformed envelope was accepted")
	}
}
