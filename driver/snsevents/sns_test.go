package snsevents

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/goforj/events/eventscore"
)

// TestDriverConstant verifies the SNS registry identifier remains stable.
func TestDriverConstant(t *testing.T) {
	driver := &Driver{}
	if got := driver.Driver(); got != eventscore.DriverSNS {
		t.Fatalf("Driver() = %q, want %q", got, eventscore.DriverSNS)
	}
}

// TestNewBuildsClientsWithDefaults verifies region defaults and client factories receive resolved settings.
func TestNewBuildsClientsWithDefaults(t *testing.T) {
	driver, err := New(Config{Endpoint: "http://127.0.0.1:4566"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if driver.snsClient == nil {
		t.Fatal("expected sns client")
	}
	if driver.sqsClient == nil {
		t.Fatal("expected sqs client")
	}
	if driver.waitTimeSeconds != defaultWaitTimeSeconds {
		t.Fatalf("waitTimeSeconds = %d, want %d", driver.waitTimeSeconds, defaultWaitTimeSeconds)
	}
	if driver.visibilityTimeout != defaultVisibilityTimeout {
		t.Fatalf("visibilityTimeout = %d, want %d", driver.visibilityTimeout, defaultVisibilityTimeout)
	}
}

// TestReadyHonorsContext verifies canceled readiness probes stop before SNS access.
func TestReadyHonorsContext(t *testing.T) {
	driver := &Driver{snsClient: stubSNSClient{}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := driver.Ready(ctx); err != context.Canceled {
		t.Fatalf("Ready() error = %v, want %v", err, context.Canceled)
	}
}

// TestPublishContextHonorsContext verifies canceled publishes do not reach SNS.
func TestPublishContextHonorsContext(t *testing.T) {
	driver := &Driver{snsClient: stubSNSClient{}, sqsClient: stubSQSClient{}, topics: make(map[string]string)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := driver.PublishContext(ctx, eventscore.Message{Topic: "orders.created", Payload: []byte("x")})
	if err != context.Canceled {
		t.Fatalf("PublishContext() error = %v, want %v", err, context.Canceled)
	}
}

// TestPublishContextPublishesToEnsuredTopic verifies publication uses the cached or newly created topic ARN.
func TestPublishContextPublishesToEnsuredTopic(t *testing.T) {
	snsClient := &recordingSNSClient{}
	driver := &Driver{
		snsClient: snsClient,
		sqsClient: stubSQSClient{},
		topics:    make(map[string]string),
	}

	err := driver.PublishContext(context.Background(), eventscore.Message{
		Topic:   "orders.created",
		Payload: []byte(`{"id":"123"}`),
	})
	if err != nil {
		t.Fatalf("PublishContext returned error: %v", err)
	}
	if snsClient.publishTopicARN != "arn:aws:sns:us-east-1:000000000000:orders-created" {
		t.Fatalf("publish topic arn = %q", snsClient.publishTopicARN)
	}
	if snsClient.publishMessage != `{"id":"123"}` {
		t.Fatalf("publish message = %q", snsClient.publishMessage)
	}
}

// TestEnsureTopicCachesTopicARN verifies repeated setup avoids duplicate SNS topic creation.
func TestEnsureTopicCachesTopicARN(t *testing.T) {
	snsClient := &recordingSNSClient{}
	driver := &Driver{
		snsClient: snsClient,
		sqsClient: stubSQSClient{},
		topics:    make(map[string]string),
	}

	first, err := driver.ensureTopic(context.Background(), "orders.created")
	if err != nil {
		t.Fatalf("first ensureTopic returned error: %v", err)
	}
	second, err := driver.ensureTopic(context.Background(), "orders.created")
	if err != nil {
		t.Fatalf("second ensureTopic returned error: %v", err)
	}
	if first != second {
		t.Fatalf("topic ARN mismatch: %q != %q", first, second)
	}
	if snsClient.createTopicCalls != 1 {
		t.Fatalf("CreateTopic calls = %d, want 1", snsClient.createTopicCalls)
	}
}

// TestSubscriptionCloseDeletesQueueAndUnsubscribes verifies SNS/SQS subscription resources are fully released.
func TestSubscriptionCloseDeletesQueueAndUnsubscribes(t *testing.T) {
	snsClient := &recordingSNSClient{}
	sqsClient := &recordingSQSClient{}
	done := make(chan struct{})
	close(done)

	sub := subscription{
		cancel:          func() {},
		done:            done,
		driver:          &Driver{snsClient: snsClient, sqsClient: sqsClient},
		queueURL:        "http://localhost:4566/queue/events",
		subscriptionARN: "arn:aws:sns:us-east-1:000000000000:sub",
	}

	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if snsClient.unsubscribeARN != "arn:aws:sns:us-east-1:000000000000:sub" {
		t.Fatalf("unsubscribe ARN = %q", snsClient.unsubscribeARN)
	}
	if sqsClient.deletedQueueURL != "http://localhost:4566/queue/events" {
		t.Fatalf("deleted queue url = %q", sqsClient.deletedQueueURL)
	}
}

// TestSanitizeName verifies generated AWS resource names contain only supported characters and lengths.
func TestSanitizeName(t *testing.T) {
	got := sanitizeName("events/orders.created", 80)
	if got != "events-orders-created" {
		t.Fatalf("sanitizeName() = %q", got)
	}
}

type stubSNSClient struct{}

// CreateTopic fails so canceled operations prove they never reach SNS setup.
func (stubSNSClient) CreateTopic(context.Context, *sns.CreateTopicInput, ...func(*sns.Options)) (*sns.CreateTopicOutput, error) {
	return nil, errors.New("unexpected CreateTopic call")
}

// ListTopics returns an empty page for readiness probes that do reach the stub.
func (stubSNSClient) ListTopics(context.Context, *sns.ListTopicsInput, ...func(*sns.Options)) (*sns.ListTopicsOutput, error) {
	return &sns.ListTopicsOutput{}, nil
}

// Publish accepts messages when a test needs a side-effect-free SNS client.
func (stubSNSClient) Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return &sns.PublishOutput{}, nil
}

// Subscribe returns an empty response for tests that do not inspect subscription metadata.
func (stubSNSClient) Subscribe(context.Context, *sns.SubscribeInput, ...func(*sns.Options)) (*sns.SubscribeOutput, error) {
	return &sns.SubscribeOutput{}, nil
}

// Unsubscribe accepts cleanup when no call recording is required.
func (stubSNSClient) Unsubscribe(context.Context, *sns.UnsubscribeInput, ...func(*sns.Options)) (*sns.UnsubscribeOutput, error) {
	return &sns.UnsubscribeOutput{}, nil
}

type stubSQSClient struct{}

// CreateQueue fails so canceled operations prove they never begin SQS setup.
func (stubSQSClient) CreateQueue(context.Context, *sqs.CreateQueueInput, ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error) {
	return nil, errors.New("unexpected CreateQueue call")
}

// DeleteMessage accepts acknowledgements when queue state is outside the assertion.
func (stubSQSClient) DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

// DeleteQueue accepts cleanup when the deleted URL is not under test.
func (stubSQSClient) DeleteQueue(context.Context, *sqs.DeleteQueueInput, ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
	return &sqs.DeleteQueueOutput{}, nil
}

// GetQueueAttributes fails so unexpected queue-policy reads remain visible.
func (stubSQSClient) GetQueueAttributes(context.Context, *sqs.GetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return nil, errors.New("unexpected GetQueueAttributes call")
}

// ReceiveMessage returns no deliveries for tests that never start the receive loop.
func (stubSQSClient) ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{}, nil
}

// SetQueueAttributes fails so unexpected queue-policy writes remain visible.
func (stubSQSClient) SetQueueAttributes(context.Context, *sqs.SetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.SetQueueAttributesOutput, error) {
	return nil, errors.New("unexpected SetQueueAttributes call")
}

type recordingSNSClient struct {
	createTopicCalls int
	publishTopicARN  string
	publishMessage   string
	unsubscribeARN   string
}

// CreateTopic records creation count and returns the ARN derived from the requested name.
func (c *recordingSNSClient) CreateTopic(_ context.Context, input *sns.CreateTopicInput, _ ...func(*sns.Options)) (*sns.CreateTopicOutput, error) {
	c.createTopicCalls++
	return &sns.CreateTopicOutput{
		TopicArn: aws.String("arn:aws:sns:us-east-1:000000000000:" + aws.ToString(input.Name)),
	}, nil
}

// ListTopics returns an empty page because recording tests exercise creation directly.
func (c *recordingSNSClient) ListTopics(context.Context, *sns.ListTopicsInput, ...func(*sns.Options)) (*sns.ListTopicsOutput, error) {
	return &sns.ListTopicsOutput{}, nil
}

// Publish records the resolved topic ARN and serialized message for assertions.
func (c *recordingSNSClient) Publish(_ context.Context, input *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
	c.publishTopicARN = aws.ToString(input.TopicArn)
	c.publishMessage = aws.ToString(input.Message)
	return &sns.PublishOutput{}, nil
}

// Subscribe returns a stable ARN so cleanup tests can verify the matching unsubscribe call.
func (c *recordingSNSClient) Subscribe(context.Context, *sns.SubscribeInput, ...func(*sns.Options)) (*sns.SubscribeOutput, error) {
	return &sns.SubscribeOutput{
		SubscriptionArn: aws.String("arn:aws:sns:us-east-1:000000000000:sub"),
	}, nil
}

// Unsubscribe records the ARN released by subscription cleanup.
func (c *recordingSNSClient) Unsubscribe(_ context.Context, input *sns.UnsubscribeInput, _ ...func(*sns.Options)) (*sns.UnsubscribeOutput, error) {
	c.unsubscribeARN = aws.ToString(input.SubscriptionArn)
	return &sns.UnsubscribeOutput{}, nil
}

type recordingSQSClient struct {
	deletedQueueURL string
}

// CreateQueue returns an empty response because queue creation details are not asserted here.
func (c *recordingSQSClient) CreateQueue(context.Context, *sqs.CreateQueueInput, ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error) {
	return &sqs.CreateQueueOutput{}, nil
}

// DeleteMessage accepts acknowledgements without adding unrelated recording state.
func (c *recordingSQSClient) DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

// DeleteQueue records the URL released by subscription cleanup.
func (c *recordingSQSClient) DeleteQueue(_ context.Context, input *sqs.DeleteQueueInput, _ ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
	c.deletedQueueURL = aws.ToString(input.QueueUrl)
	return &sqs.DeleteQueueOutput{}, nil
}

// GetQueueAttributes returns an empty policy view when setup details are not asserted.
func (c *recordingSQSClient) GetQueueAttributes(context.Context, *sqs.GetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return &sqs.GetQueueAttributesOutput{}, nil
}

// ReceiveMessage returns no deliveries because cleanup tests do not run a consumer loop.
func (c *recordingSQSClient) ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{}, nil
}

// SetQueueAttributes accepts policy setup without adding assertions unrelated to cleanup.
func (c *recordingSQSClient) SetQueueAttributes(context.Context, *sqs.SetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.SetQueueAttributesOutput, error) {
	return &sqs.SetQueueAttributesOutput{}, nil
}
