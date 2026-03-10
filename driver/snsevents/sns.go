package snsevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/goforj/events/eventscore"
)

var nextSubscriptionID atomic.Uint64

var queueNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

const (
	defaultRegion            = "us-east-1"
	defaultWaitTimeSeconds   = int32(1)
	defaultVisibilityTimeout = int32(30)
	maxQueueNameLen          = 80
)

type snsAPI interface {
	CreateTopic(context.Context, *sns.CreateTopicInput, ...func(*sns.Options)) (*sns.CreateTopicOutput, error)
	ListTopics(context.Context, *sns.ListTopicsInput, ...func(*sns.Options)) (*sns.ListTopicsOutput, error)
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
	Subscribe(context.Context, *sns.SubscribeInput, ...func(*sns.Options)) (*sns.SubscribeOutput, error)
	Unsubscribe(context.Context, *sns.UnsubscribeInput, ...func(*sns.Options)) (*sns.UnsubscribeOutput, error)
}

type sqsAPI interface {
	CreateQueue(context.Context, *sqs.CreateQueueInput, ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error)
	DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	DeleteQueue(context.Context, *sqs.DeleteQueueInput, ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
	GetQueueAttributes(context.Context, *sqs.GetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	SetQueueAttributes(context.Context, *sqs.SetQueueAttributesInput, ...func(*sqs.Options)) (*sqs.SetQueueAttributesOutput, error)
}

// Driver is an SNS-backed events transport using SQS queues for subscriptions.
// @group Drivers
//
// Example: keep an SNS driver reference
//
//	var driver *snsevents.Driver
//	fmt.Println(driver == nil)
//	// Output: true
type Driver struct {
	snsClient snsAPI
	sqsClient sqsAPI

	topicNamePrefix   string
	queueNamePrefix   string
	waitTimeSeconds   int32
	visibilityTimeout int32
	topicsMu          sync.RWMutex
	topics            map[string]string
}

// Config configures SNS transport construction.
// @group Config
//
// Example: define SNS driver config
//
//	cfg := snsevents.Config{
//		Region:   "us-east-1",
//		Endpoint: "http://127.0.0.1:4566",
//	}
//	_ = cfg
//
// Example: define SNS driver config with all fields
//
//	cfg := snsevents.Config{
//		Region:            "us-east-1",
//		Endpoint:          "http://127.0.0.1:4566", // default: "" uses normal AWS resolution
//		SNSClient:         nil,                      // default: nil creates a client from Region and Endpoint
//		SQSClient:         nil,                      // default: nil creates a client from Region and Endpoint
//		TopicNamePrefix:   "events-",                // default: ""
//		QueueNamePrefix:   "events-",                // default: ""
//		WaitTimeSeconds:   1,                        // default: 1
//		VisibilityTimeout: 30,                       // default: 30
//	}
//	_ = cfg
type Config struct {
	Region            string
	Endpoint          string
	SNSClient         *sns.Client
	SQSClient         *sqs.Client
	TopicNamePrefix   string
	QueueNamePrefix   string
	WaitTimeSeconds   int32
	VisibilityTimeout int32
}

// New constructs an SNS-backed driver.
// @group Driver Constructors
//
// Example: construct an SNS driver
//
//	driver, _ := snsevents.New(snsevents.Config{
//		Region:   "us-east-1",
//		Endpoint: "http://127.0.0.1:4566",
//	})
//	_ = driver
func New(cfg Config) (*Driver, error) {
	if cfg.SNSClient == nil || cfg.SQSClient == nil {
		awsCfg, err := loadAWSConfig(cfg)
		if err != nil {
			return nil, err
		}
		if cfg.SNSClient == nil {
			cfg.SNSClient = sns.NewFromConfig(awsCfg)
		}
		if cfg.SQSClient == nil {
			cfg.SQSClient = sqs.NewFromConfig(awsCfg)
		}
	}
	waitTimeSeconds := cfg.WaitTimeSeconds
	if waitTimeSeconds <= 0 {
		waitTimeSeconds = defaultWaitTimeSeconds
	}
	visibilityTimeout := cfg.VisibilityTimeout
	if visibilityTimeout <= 0 {
		visibilityTimeout = defaultVisibilityTimeout
	}
	return &Driver{
		snsClient:         cfg.SNSClient,
		sqsClient:         cfg.SQSClient,
		topicNamePrefix:   cfg.TopicNamePrefix,
		queueNamePrefix:   cfg.QueueNamePrefix,
		waitTimeSeconds:   waitTimeSeconds,
		visibilityTimeout: visibilityTimeout,
		topics:            make(map[string]string),
	}, nil
}

func loadAWSConfig(cfg Config) (aws.Config, error) {
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = defaultRegion
	}
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if strings.TrimSpace(cfg.Endpoint) != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, resolvedRegion string, _ ...interface{}) (aws.Endpoint, error) {
			switch service {
			case sns.ServiceID, sqs.ServiceID:
				return aws.Endpoint{
					URL:               cfg.Endpoint,
					PartitionID:       "aws",
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			default:
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}
		})
		options = append(options,
			awsconfig.WithEndpointResolverWithOptions(resolver),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
	}
	return awsconfig.LoadDefaultConfig(context.Background(), options...)
}

// Driver reports the active backend kind.
// @group Drivers
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverSNS
}

// Ready checks SNS connectivity.
// @group Drivers
func (d *Driver) Ready(ctx context.Context) error {
	var err error
	ctx, err = normalizeContext(ctx)
	if err != nil {
		return err
	}
	_, err = d.snsClient.ListTopics(ctx, &sns.ListTopicsInput{})
	return err
}

// PublishContext publishes a topic payload to SNS.
// @group Drivers
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	var err error
	ctx, err = normalizeContext(ctx)
	if err != nil {
		return err
	}
	topicARN, err := d.ensureTopic(ctx, msg.Topic)
	if err != nil {
		return err
	}
	_, err = d.snsClient.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(string(msg.Payload)),
	})
	return err
}

// SubscribeContext subscribes to an SNS topic through a dedicated SQS queue.
// @group Drivers
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	var err error
	ctx, err = normalizeContext(ctx)
	if err != nil {
		return nil, err
	}
	topicARN, err := d.ensureTopic(ctx, topic)
	if err != nil {
		return nil, err
	}
	queueName := d.queueName(topic)
	createQueueOut, err := d.sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]string{
			string(sqstypes.QueueAttributeNameReceiveMessageWaitTimeSeconds): fmt.Sprintf("%d", d.waitTimeSeconds),
			string(sqstypes.QueueAttributeNameVisibilityTimeout):             fmt.Sprintf("%d", d.visibilityTimeout),
		},
	})
	if err != nil {
		return nil, err
	}
	queueURL := aws.ToString(createQueueOut.QueueUrl)
	if queueURL == "" {
		return nil, errors.New("snsevents: create queue returned empty QueueUrl")
	}

	attrsOut, err := d.sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		_ = d.deleteQueue(context.Background(), queueURL)
		return nil, err
	}
	queueARN := attrsOut.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]
	if queueARN == "" {
		_ = d.deleteQueue(context.Background(), queueURL)
		return nil, errors.New("snsevents: queue ARN was empty")
	}

	if err := d.allowTopicToSend(ctx, queueURL, queueARN, topicARN); err != nil {
		_ = d.deleteQueue(context.Background(), queueURL)
		return nil, err
	}

	subscribeOut, err := d.snsClient.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn: aws.String(topicARN),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String(queueARN),
		Attributes: map[string]string{
			"RawMessageDelivery": "true",
		},
		ReturnSubscriptionArn: true,
	})
	if err != nil {
		_ = d.deleteQueue(context.Background(), queueURL)
		return nil, err
	}

	workerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.pollQueue(workerCtx, queueURL, topic, handler)
	}()

	return subscription{
		cancel:          cancel,
		done:            done,
		driver:          d,
		queueURL:        queueURL,
		subscriptionARN: aws.ToString(subscribeOut.SubscriptionArn),
	}, nil
}

// Close releases driver-owned resources.
// @group Lifecycle
func (d *Driver) Close() error {
	return nil
}

func (d *Driver) ensureTopic(ctx context.Context, topic string) (string, error) {
	var err error
	ctx, err = normalizeContext(ctx)
	if err != nil {
		return "", err
	}
	if arn, ok := d.cachedTopic(topic); ok {
		return arn, nil
	}

	d.topicsMu.Lock()
	defer d.topicsMu.Unlock()
	if arn, ok := d.topics[topic]; ok {
		return arn, nil
	}
	createOut, err := d.snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String(d.topicName(topic)),
	})
	if err != nil {
		return "", err
	}
	arn := aws.ToString(createOut.TopicArn)
	if arn == "" {
		return "", errors.New("snsevents: create topic returned empty TopicArn")
	}
	d.topics[topic] = arn
	return arn, nil
}

func (d *Driver) cachedTopic(topic string) (string, bool) {
	d.topicsMu.RLock()
	defer d.topicsMu.RUnlock()
	arn, ok := d.topics[topic]
	return arn, ok
}

func (d *Driver) topicName(topic string) string {
	return sanitizeName(d.topicNamePrefix+topic, maxQueueNameLen)
}

func (d *Driver) queueName(topic string) string {
	name := fmt.Sprintf("%s%s-%d", d.queueNamePrefix, topic, nextSubscriptionID.Add(1))
	return sanitizeName(name, maxQueueNameLen)
}

func sanitizeName(value string, maxLen int) string {
	value = queueNameSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		value = "events"
	}
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	return value
}

func (d *Driver) allowTopicToSend(ctx context.Context, queueURL, queueARN, topicARN string) error {
	policyBytes, err := json.Marshal(queuePolicy{
		Version: "2012-10-17",
		Statement: []queuePolicyStatement{
			{
				Sid:       "AllowSNSSend",
				Effect:    "Allow",
				Principal: map[string]string{"Service": "sns.amazonaws.com"},
				Action:    "sqs:SendMessage",
				Resource:  queueARN,
				Condition: queuePolicyCondition{
					ArnEquals: map[string]string{"aws:SourceArn": topicARN},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	_, err = d.sqsClient.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(queueURL),
		Attributes: map[string]string{
			string(sqstypes.QueueAttributeNamePolicy): string(policyBytes),
		},
	})
	return err
}

func (d *Driver) pollQueue(ctx context.Context, queueURL, topic string, handler eventscore.MessageHandler) {
	for {
		if ctx.Err() != nil {
			return
		}
		pollCtx, cancel := context.WithTimeout(ctx, time.Duration(d.waitTimeSeconds+1)*time.Second)
		out, err := d.sqsClient.ReceiveMessage(pollCtx, &sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(queueURL),
			MaxNumberOfMessages:   1,
			WaitTimeSeconds:       d.waitTimeSeconds,
			VisibilityTimeout:     d.visibilityTimeout,
			MessageAttributeNames: []string{"All"},
		})
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		for _, message := range out.Messages {
			_ = handler(context.Background(), eventscore.Message{
				Topic:   topic,
				Payload: []byte(aws.ToString(message.Body)),
			})
			if message.ReceiptHandle != nil {
				_, _ = d.sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueURL),
					ReceiptHandle: message.ReceiptHandle,
				})
			}
		}
	}
}

func (d *Driver) deleteQueue(ctx context.Context, queueURL string) error {
	var err error
	ctx, err = normalizeContext(ctx)
	if err != nil {
		return err
	}
	_, err = d.sqsClient.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	return err
}

func normalizeContext(ctx context.Context) (context.Context, error) {
	if ctx == nil {
		return context.Background(), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return ctx, nil
}

type subscription struct {
	cancel          context.CancelFunc
	done            <-chan struct{}
	driver          *Driver
	queueURL        string
	subscriptionARN string
}

func (s subscription) Close() error {
	s.cancel()
	<-s.done

	var result error
	if s.subscriptionARN != "" && s.subscriptionARN != "pending confirmation" {
		_, err := s.driver.snsClient.Unsubscribe(context.Background(), &sns.UnsubscribeInput{
			SubscriptionArn: aws.String(s.subscriptionARN),
		})
		result = errors.Join(result, err)
	}
	result = errors.Join(result, s.driver.deleteQueue(context.Background(), s.queueURL))
	return result
}

type queuePolicy struct {
	Version   string                 `json:"Version"`
	Statement []queuePolicyStatement `json:"Statement"`
}

type queuePolicyStatement struct {
	Sid       string               `json:"Sid"`
	Effect    string               `json:"Effect"`
	Principal map[string]string    `json:"Principal"`
	Action    string               `json:"Action"`
	Resource  string               `json:"Resource"`
	Condition queuePolicyCondition `json:"Condition"`
}

type queuePolicyCondition struct {
	ArnEquals map[string]string `json:"ArnEquals"`
}
