package redisevents

import (
	"bytes"
	"context"
	"testing"

	"github.com/goforj/events/eventscore"
	"github.com/redis/go-redis/v9"
)

// TestRawMessageCompatibility preserves arbitrary binary bytes when framing is not explicitly enabled.
func TestRawMessageCompatibility(t *testing.T) {
	for _, body := range [][]byte{nil, {}, {0, 1, 2, 255}, append(append([]byte(nil), messagePrefix...), []byte("null")...), append(append([]byte(nil), messagePrefix...), []byte("invalid")...), append(append([]byte(nil), messagePrefix...), []byte(`{"headers":{},"payload":"WA=="}`)...)} {
		encoded, err := encodeMessage(eventscore.Message{Payload: body}, false)
		if err != nil || !bytes.Equal(encoded, body) {
			t.Fatal("raw publication changed")
		}
		decoded, err := decodeMessage(encoded, false)
		if err != nil || !bytes.Equal(decoded.Payload, body) || len(decoded.Headers) != 0 {
			t.Fatal("raw delivery changed")
		}
		escaped, err := encodeMessage(eventscore.Message{Payload: body}, true)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err = decodeMessage(escaped, true)
		if err != nil || !bytes.Equal(decoded.Payload, body) || len(decoded.Headers) != 0 {
			t.Fatal("opted-in binary round trip changed")
		}
	}
}

// TestMessageHeaderFraming preserves business payloads, including nil and empty bodies, on opted-in transports.
func TestMessageHeaderFraming(t *testing.T) {
	for _, body := range [][]byte{nil, {}, {0, 1, 2, 255, '{', '}'}} {
		encoded, err := encodeMessage(eventscore.Message{Payload: body, Headers: map[string]string{"traceparent": "producer", "tracestate": "vendor=value"}}, true)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeMessage(encoded, true)
		if err != nil || !bytes.Equal(decoded.Payload, body) || decoded.Headers["traceparent"] != "producer" || decoded.Headers["tracestate"] != "vendor=value" {
			t.Fatalf("header round trip: %+v %v", decoded, err)
		}
	}
}

// TestMessageFramingRejectsMalformedEnvelopes avoids treating null or incomplete JSON as successful empty delivery.
func TestMessageFramingRejectsMalformedEnvelopes(t *testing.T) {
	for _, body := range []string{"invalid", "null", "{}", `{"headers":{}}`, `{"payload":42}`, `{"payload":"not base64"}`} {
		frame := append(append([]byte(nil), messagePrefix...), []byte(body)...)
		if _, err := decodeMessage(frame, true); err == nil {
			t.Fatalf("accepted invalid envelope %s", body)
		}
	}
}

// TestMessageHeadersRequireOptIn rejects metadata before attempting Redis I/O and copies configuration on both constructor paths.
func TestMessageHeadersRequireOptIn(t *testing.T) {
	for _, injected := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			cfg := Config{Addr: "127.0.0.1:1", EnableHeaders: enabled}
			if injected {
				cfg.Client = redis.NewClient(&redis.Options{Addr: cfg.Addr})
			}
			driver, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if driver.enableHeaders != enabled {
				t.Fatal("constructor lost framing option")
			}
			if !enabled {
				if err := driver.PublishContext(context.Background(), eventscore.Message{Headers: map[string]string{"traceparent": "value"}}); err == nil || err.Error() != "redisevents: message headers require Config.EnableHeaders" {
					t.Fatalf("disabled headers error=%v", err)
				}
			}
			if err := driver.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestMalformedHeadersPreservePayload discards invalid optional metadata independently from valid business delivery.
func TestMalformedHeadersPreservePayload(t *testing.T) {
	for _, headers := range []string{`42`, `false`, `"invalid"`, `[1,2]`, `{"traceparent":42}`} {
		frame := append(append([]byte(nil), messagePrefix...), []byte(`{"headers":`+headers+`,"payload":"ZXhhY3Q="}`)...)
		message, err := decodeMessage(frame, true)
		if err != nil || string(message.Payload) != "exact" || len(message.Headers) != 0 {
			t.Fatalf("headers=%s message=%+v err=%v", headers, message, err)
		}
	}
}

// TestInvalidHeaderUTF8PreservesPayload discards malformed metadata before JSON can repair its bytes.
func TestInvalidHeaderUTF8PreservesPayload(t *testing.T) {
	for _, headers := range []string{
		`{"traceparent":"producer","extra":"` + string([]byte{255}) + `"}`,
		`{"traceparent":"producer","` + string([]byte{255}) + `":"extra"}`,
	} {
		frame := append(append([]byte(nil), messagePrefix...), []byte(`{"headers":`+headers+`,"payload":"ZXhhY3Q="}`)...)
		message, err := decodeMessage(frame, true)
		if err != nil || string(message.Payload) != "exact" || len(message.Headers) != 0 {
			t.Fatalf("headers=%q message=%+v err=%v", headers, message, err)
		}
	}
}

// TestValidUnicodeHeadersPreservePayload distinguishes real replacement characters from invalid encoded bytes.
func TestValidUnicodeHeadersPreservePayload(t *testing.T) {
	encoded, err := encodeMessage(eventscore.Message{Payload: []byte("exact"), Headers: map[string]string{"traceparent": "producer", "extra": "é\uFFFD"}}, true)
	if err != nil {
		t.Fatal(err)
	}
	message, err := decodeMessage(encoded, true)
	if err != nil || string(message.Payload) != "exact" || message.Headers["traceparent"] != "producer" || message.Headers["extra"] != "é\uFFFD" {
		t.Fatalf("message=%+v err=%v", message, err)
	}
}
