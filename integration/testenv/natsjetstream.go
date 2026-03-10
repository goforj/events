package testenv

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NATSJetStreamContainer holds the started NATS JetStream container and connection details.
type NATSJetStreamContainer struct {
	Container testcontainers.Container
	URL       string
}

// StartNATSJetStream starts a disposable NATS container with JetStream enabled.
func StartNATSJetStream(ctx context.Context) (*NATSJetStreamContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        natsImage,
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js"},
		WaitingFor:   wait.ForListeningPort("4222/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}
	port, err := container.MappedPort(ctx, "4222/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}
	return &NATSJetStreamContainer{
		Container: container,
		URL:       fmt.Sprintf("nats://%s:%s", host, port.Port()),
	}, nil
}
