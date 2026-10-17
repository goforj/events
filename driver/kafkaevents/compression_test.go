package kafkaevents

import (
	"bytes"
	"io"
	"testing"

	"github.com/segmentio/kafka-go"
)

// TestKafkaZstdCodecRoundTrip preserves Kafka's compressed message format across dependency updates.
func TestKafkaZstdCodecRoundTrip(t *testing.T) {
	const payload = "a compressed Kafka message"
	var compressed bytes.Buffer
	codec := kafka.Zstd.Codec()
	writer := codec.NewWriter(&compressed)
	if _, err := writer.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader := codec.NewReader(&compressed)
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Fatalf("decoded message = %q, want %q", got, payload)
	}
}
