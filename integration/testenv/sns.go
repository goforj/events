package testenv

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const localstackImage = "localstack/localstack:3.8.1"

// SNSContainer holds the started LocalStack container and connection details.
type SNSContainer struct {
	Container testcontainers.Container
	Endpoint  string
	Region    string
}

// StartSNS starts a disposable LocalStack SNS/SQS container for integration tests.
func StartSNS(ctx context.Context) (*SNSContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        localstackImage,
		ExposedPorts: []string{"4566/tcp"},
		Env: map[string]string{
			"SERVICES":       "sns,sqs",
			"DEFAULT_REGION": defaultSNSRegion,
		},
		WaitingFor: wait.ForListeningPort("4566/tcp").WithStartupTimeout(60 * time.Second),
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
	port, err := container.MappedPort(ctx, "4566/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}
	return &SNSContainer{
		Container: container,
		Endpoint:  fmt.Sprintf("http://%s:%s", host, port.Port()),
		Region:    defaultSNSRegion,
	}, nil
}

const defaultSNSRegion = "us-east-1"
