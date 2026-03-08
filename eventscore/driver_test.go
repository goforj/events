package eventscore

import "testing"

func TestDriverConstants(t *testing.T) {
	tests := map[string]Driver{
		"sync":      DriverSync,
		"null":      DriverNull,
		"nats":      DriverNATS,
		"redis":     DriverRedis,
		"kafka":     DriverKafka,
		"gcppubsub": DriverGCPPubSub,
		"sqs":       DriverSQS,
	}

	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("driver constant mismatch: got=%q want=%q", got, want)
		}
	}
}
