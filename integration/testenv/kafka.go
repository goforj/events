package testenv

import (
	"context"

	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

const kafkaImage = "confluentinc/confluent-local:7.5.0"

// KafkaContainer holds the started Kafka container and connection details.
type KafkaContainer struct {
	Container *tckafka.KafkaContainer
	Brokers   []string
}

// StartKafka starts a disposable Kafka container for integration tests.
func StartKafka(ctx context.Context) (*KafkaContainer, error) {
	container, err := tckafka.Run(ctx, kafkaImage, tckafka.WithClusterID("events-test-cluster"))
	if err != nil {
		return nil, err
	}
	brokers, err := container.Brokers(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}
	return &KafkaContainer{
		Container: container,
		Brokers:   brokers,
	}, nil
}
