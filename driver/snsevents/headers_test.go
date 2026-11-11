package snsevents

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/goforj/events/eventscore"
)

// TestCarrierHeaders exercises native string attributes without changing the message body.
func TestCarrierHeaders(t *testing.T) {
	if publishHeaders(nil) != nil || receivedHeaders(nil) != nil {
		t.Fatal("empty metadata changed the ordinary path")
	}
	native := publishHeaders(map[string]string{"traceparent": "producer"})
	if aws.ToString(native["traceparent"].StringValue) != "producer" || aws.ToString(native["traceparent"].DataType) != "String" {
		t.Fatal(native)
	}
	headers := receivedHeaders(map[string]types.MessageAttributeValue{"traceparent": {StringValue: aws.String("producer")}, "binary": {BinaryValue: []byte("ignored")}})
	if headers["traceparent"] != "producer" || len(headers) != 1 {
		t.Fatal(headers)
	}
}

// TestPublishHeadersInvalidAttributes verifies any invalid optional attribute drops the complete carrier.
func TestPublishHeadersInvalidAttributes(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty name", key: "", value: "value"},
		{name: "long name", key: strings.Repeat("a", 257), value: "value"},
		{name: "leading period", key: ".trace", value: "value"},
		{name: "trailing period", key: "trace.", value: "value"},
		{name: "consecutive periods", key: "trace..parent", value: "value"},
		{name: "aws prefix", key: "AwS.trace", value: "value"},
		{name: "amazon prefix", key: "aMaZoN.trace", value: "value"},
		{name: "space", key: "correlation id", value: "value"},
		{name: "punctuation", key: "trace:parent", value: "value"},
		{name: "unicode name", key: "tracé", value: "value"},
		{name: "empty value", key: "emptyvalue", value: ""},
		{name: "invalid utf8", key: "trace", value: string([]byte{0xff})},
		{name: "surrogate encoding", key: "trace", value: string([]byte{0xed, 0xa0, 0x80})},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := map[string]string{"traceparent": "valid", test.key: test.value}
			if got := publishHeaders(headers); got != nil {
				t.Fatalf("publishHeaders() = %v, want omitted carrier", got)
			}
			client := &recordingSNSClient{}
			driver := &Driver{snsClient: client, topics: make(map[string]string)}
			const payload = `{"id":"123","message":"unchanged"}`
			if err := driver.PublishContext(context.Background(), eventscore.Message{
				Topic: "orders.created", Payload: []byte(payload), Headers: headers,
			}); err != nil {
				t.Fatalf("PublishContext() error = %v", err)
			}
			if client.publishTopicARN == "" || client.publishMessage != payload || client.publishAttributes != nil {
				t.Fatalf("publication changed: topic %q, body %q, attributes %v", client.publishTopicARN, client.publishMessage, client.publishAttributes)
			}
		})
	}
}

// TestPublishHeadersValidSnapshot preserves SNS-compatible names and UTF-8 values without JSON interpretation.
func TestPublishHeadersValidSnapshot(t *testing.T) {
	for _, name := range []string{"a", "AZaz09_-", "trace.parent", strings.Repeat("a", 256), "AWS", "Amazon", "aws-trace", "amazon_trace"} {
		t.Run(name, func(t *testing.T) {
			if got := publishHeaders(map[string]string{name: "value"}); len(got) != 1 {
				t.Fatalf("valid name %q omitted", name)
			}
		})
	}
	headers := map[string]string{
		"plain": "producer", "unicode": "café 世界 😀", "whitespace": " \t\n",
		"quoted": `"quoted"`, "json": `{"id":1}`, "escape": `\uD800`, "replacement": "\uFFFD",
	}
	client := &recordingSNSClient{}
	driver := &Driver{snsClient: client, topics: make(map[string]string)}
	if err := driver.PublishContext(context.Background(), eventscore.Message{
		Topic: "orders.created", Payload: []byte("exact"), Headers: headers,
	}); err != nil {
		t.Fatal(err)
	}
	if client.publishMessage != "exact" || len(client.publishAttributes) != len(headers) {
		t.Fatal("valid publication changed")
	}
	for key, value := range headers {
		attribute := client.publishAttributes[key]
		if aws.ToString(attribute.DataType) != "String" || aws.ToString(attribute.StringValue) != value {
			t.Fatalf("attribute %q = %v, want String %q", key, attribute, value)
		}
	}
	headers["plain"] = "changed"
	delete(headers, "unicode")
	if aws.ToString(client.publishAttributes["plain"].StringValue) != "producer" || aws.ToString(client.publishAttributes["unicode"].StringValue) != "café 世界 😀" {
		t.Fatal("published attributes alias the source carrier")
	}
	if got := publishHeaders(map[string]string{}); got != nil {
		t.Fatal("empty metadata was not omitted")
	}
}
