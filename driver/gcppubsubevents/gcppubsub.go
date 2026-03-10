package gcppubsubevents

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	gpubsub "cloud.google.com/go/pubsub"
	"github.com/goforj/events/eventscore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var nextSubscriptionID atomic.Uint64

// Driver is a Google Pub/Sub-backed events transport.
// @group Drivers
//
// Example: keep a Google Pub/Sub driver reference
//
//	var driver *gcppubsubevents.Driver
//	fmt.Println(driver == nil)
//	// Output: true
type Driver struct {
	projectID  string
	client     *gpubsub.Client
	conn       *grpc.ClientConn
	ownsClient bool

	topicsMu sync.RWMutex
	topics   map[string]*gpubsub.Topic
}

// Config configures Google Pub/Sub transport construction.
// @group Config
//
// Example: define Google Pub/Sub driver config
//
//	cfg := gcppubsubevents.Config{
//		ProjectID: "events-project",
//		URI:       "127.0.0.1:8085",
//	}
//	_ = cfg
//
// Example: define Google Pub/Sub driver config with all fields
//
//	cfg := gcppubsubevents.Config{
//		ProjectID: "events-project",
//		URI:       "127.0.0.1:8085", // default: "" is invalid unless Client is provided
//		Client:    nil,              // default: nil creates a client from ProjectID and URI
//	}
//	_ = cfg
type Config struct {
	ProjectID string
	URI       string
	Client    *gpubsub.Client
}

const (
	defaultPublishDelayThreshold = 1 * time.Millisecond
	defaultPublishCountThreshold = 1
)

// New constructs a Google Pub/Sub-backed driver.
// @group Driver Constructors
//
// Example: construct a Google Pub/Sub driver
//
//	driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
//		ProjectID: "events-project",
//		URI:       "127.0.0.1:8085",
//	})
//	_ = driver
func New(ctx context.Context, cfg Config) (*Driver, error) {
	if cfg.Client != nil {
		if cfg.ProjectID == "" {
			return nil, errors.New("gcppubsubevents: ProjectID is required")
		}
		return &Driver{
			projectID: cfg.ProjectID,
			client:    cfg.Client,
			topics:    make(map[string]*gpubsub.Topic),
		}, nil
	}
	if cfg.ProjectID == "" {
		return nil, errors.New("gcppubsubevents: ProjectID is required")
	}
	if cfg.URI == "" {
		return nil, errors.New("gcppubsubevents: URI is required")
	}

	conn, err := grpc.NewClient(cfg.URI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client, err := gpubsub.NewClient(ctx, cfg.ProjectID, option.WithGRPCConn(conn))
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Driver{
		projectID:  cfg.ProjectID,
		client:     client,
		conn:       conn,
		ownsClient: true,
		topics:     make(map[string]*gpubsub.Topic),
	}, nil
}

// Driver reports the active backend kind.
// @group Drivers
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverGCPPubSub
}

// Ready checks Google Pub/Sub connectivity.
// @group Drivers
func (d *Driver) Ready(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	it := d.client.Topics(ctx)
	_, err := it.Next()
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	// Iterator completion is still a successful connectivity check.
	return nil
}

// PublishContext publishes a topic payload to Google Pub/Sub.
// @group Drivers
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	topic, err := d.ensureTopic(ctx, msg.Topic)
	if err != nil {
		return err
	}
	result := topic.Publish(ctx, &gpubsub.Message{Data: msg.Payload})
	_, err = result.Get(ctx)
	return err
}

// SubscribeContext subscribes to a Google Pub/Sub topic and forwards messages.
// @group Drivers
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	pubTopic, err := d.ensureTopic(ctx, topic)
	if err != nil {
		return nil, err
	}
	subID := fmt.Sprintf("events-%s-%d", topic, nextSubscriptionID.Add(1))
	sub, err := d.client.CreateSubscription(ctx, subID, gpubsub.SubscriptionConfig{Topic: pubTopic})
	if err != nil {
		return nil, err
	}
	sub.ReceiveSettings.NumGoroutines = 1

	workerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		err := sub.Receive(workerCtx, func(_ context.Context, msg *gpubsub.Message) {
			_ = handler(context.Background(), eventscore.Message{
				Topic:   topic,
				Payload: msg.Data,
			})
			msg.Ack()
		})
		done <- err
	}()

	return subscription{
		cancel: cancel,
		done:   done,
		sub:    sub,
	}, nil
}

// Close closes the underlying Pub/Sub client.
// @group Drivers
func (d *Driver) Close() error {
	d.stopTopics()
	if d.ownsClient && d.client != nil {
		if err := d.client.Close(); err != nil {
			return err
		}
	}
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

func (d *Driver) ensureTopic(ctx context.Context, topic string) (*gpubsub.Topic, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if pubTopic, ok := d.cachedTopic(topic); ok {
		return pubTopic, nil
	}

	d.topicsMu.Lock()
	defer d.topicsMu.Unlock()
	if pubTopic, ok := d.topics[topic]; ok {
		return pubTopic, nil
	}

	pubTopic := d.newTopic(topic)
	exists, err := pubTopic.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		d.topics[topic] = pubTopic
		return pubTopic, nil
	}
	created, err := d.client.CreateTopic(ctx, topic)
	if err != nil {
		return nil, err
	}
	d.configureTopic(created)
	d.topics[topic] = created
	return created, nil
}

func (d *Driver) cachedTopic(topic string) (*gpubsub.Topic, bool) {
	d.topicsMu.RLock()
	defer d.topicsMu.RUnlock()
	pubTopic, ok := d.topics[topic]
	return pubTopic, ok
}

func (d *Driver) newTopic(topic string) *gpubsub.Topic {
	pubTopic := d.client.Topic(topic)
	d.configureTopic(pubTopic)
	return pubTopic
}

func (d *Driver) configureTopic(topic *gpubsub.Topic) {
	topic.PublishSettings.DelayThreshold = defaultPublishDelayThreshold
	topic.PublishSettings.CountThreshold = defaultPublishCountThreshold
}

func (d *Driver) stopTopics() {
	d.topicsMu.Lock()
	defer d.topicsMu.Unlock()
	for key, topic := range d.topics {
		topic.Stop()
		delete(d.topics, key)
	}
}

type subscription struct {
	cancel context.CancelFunc
	done   <-chan error
	sub    *gpubsub.Subscription
}

func (s subscription) Close() error {
	s.cancel()
	err := <-s.done
	var deleteErr error
	if s.sub != nil {
		deleteCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		deleteErr = s.sub.Delete(deleteCtx)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return deleteErr
}
