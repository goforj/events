package kafkaevents

import (
	"github.com/segmentio/kafka-go"
	"strings"
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
	repeated := receivedHeaders([]kafka.Header{{Key: "traceparent", Value: []byte("first")}, {Key: "traceparent", Value: []byte("second")}})
	if repeated["traceparent"] != "first,second" {
		t.Fatal("ambiguous parents were hidden")
	}
}

// TestReceivedHeadersManyDuplicates preserves order and empty values without repeatedly copying the accumulated field.
func TestReceivedHeadersManyDuplicates(t *testing.T) {
	const count = 20000
	headers := make([]kafka.Header, 0, count+2)
	expected := make([]string, 0, count)
	headers = append(headers, kafka.Header{Key: "unique", Value: []byte("kept")})
	for index := range count {
		value := "parent"
		if index%3 == 0 {
			value = ""
		}
		headers = append(headers, kafka.Header{Key: "traceparent", Value: []byte(value)})
		expected = append(expected, value)
	}
	headers = append(headers, kafka.Header{Key: "tracestate", Value: []byte("vendor=value")})
	received := receivedHeaders(headers)
	if received["traceparent"] != strings.Join(expected, ",") || received["unique"] != "kept" || received["tracestate"] != "vendor=value" {
		t.Fatal("duplicate aggregation changed carrier contents")
	}
}
