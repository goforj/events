package testenv

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const natsImage = "nats:2.11-alpine"

// NATSContainer holds the started NATS container and connection details.
type NATSContainer struct {
	Container testcontainers.Container
	URL       string
}

// StartNATS starts a disposable NATS container for integration tests.
func StartNATS(ctx context.Context) (*NATSContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        natsImage,
		ExposedPorts: []string{"4222/tcp"},
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
	return &NATSContainer{
		Container: container,
		URL:       fmt.Sprintf("nats://%s:%s", host, port.Port()),
	}, nil
}
