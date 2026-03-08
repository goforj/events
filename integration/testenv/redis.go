package testenv

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const redisImage = "redis:7-alpine"

// RedisContainer holds the started Redis container and connection details.
type RedisContainer struct {
	Container testcontainers.Container
	Addr      string
}

// StartRedis starts a disposable Redis container for integration tests.
func StartRedis(ctx context.Context) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        redisImage,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
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
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}
	return &RedisContainer{
		Container: container,
		Addr:      fmt.Sprintf("%s:%s", host, port.Port()),
	}, nil
}
