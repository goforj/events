package natsjetstreamevents

import (
	"github.com/nats-io/nats.go"
	"testing"
)

// TestCarrierHeaders exercises native fields, repeated values, and the ordinary header-free path.
func TestCarrierHeaders(t *testing.T) {
	if publishHeaders(nil) != nil || receivedHeaders(nil) != nil {
		t.Fatal("empty metadata changed the ordinary path")
	}
	headers := receivedHeaders(publishHeaders(map[string]string{"traceparent": "producer", "tracestate": "vendor=value"}))
	if headers["traceparent"] != "producer" || headers["tracestate"] != "vendor=value" {
		t.Fatal(headers)
	}
	repeated := receivedHeaders(nats.Header{"traceparent": []string{"first", "second"}})
	if repeated["traceparent"] != "first,second" {
		t.Fatal("ambiguous parents were hidden")
	}
}
