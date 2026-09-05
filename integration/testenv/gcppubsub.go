package testenv

import (
	"context"

	tcpubsub "github.com/testcontainers/testcontainers-go/modules/gcloud/pubsub"
)

const gcppubsubImage = "gcr.io/google.com/cloudsdktool/cloud-sdk@sha256:9e3cddf9cd9fa726c38c7320904ef601ce4a7c815c8c11c24004146d85a195c1"

// GCPPubSubContainer holds the started Pub/Sub emulator and connection details.
type GCPPubSubContainer struct {
	Container *tcpubsub.Container
	ProjectID string
	URI       string
}

// StartGCPPubSub starts a disposable Google Pub/Sub emulator container for integration tests.
func StartGCPPubSub(ctx context.Context) (*GCPPubSubContainer, error) {
	container, err := tcpubsub.Run(ctx, gcppubsubImage, tcpubsub.WithProjectID("events-pubsub-project"))
	if err != nil {
		return nil, err
	}
	return &GCPPubSubContainer{
		Container: container,
		ProjectID: container.ProjectID(),
		URI:       container.URI(),
	}, nil
}
