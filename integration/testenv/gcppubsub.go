package testenv

import (
	"context"

	tcpubsub "github.com/testcontainers/testcontainers-go/modules/gcloud/pubsub"
)

const gcppubsubImage = "gcr.io/google.com/cloudsdktool/cloud-sdk:367.0.0-emulators"

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
