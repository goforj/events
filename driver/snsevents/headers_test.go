package snsevents

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"testing"
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
