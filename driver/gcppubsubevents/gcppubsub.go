package gcppubsubevents

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	gpubsub "cloud.google.com/go/pubsub"
	"github.com/goforj/events/eventscore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var nextSubscriptionID atomic.Uint64

// Driver is a Google Pub/Sub-backed events transport.
type Driver struct {
	projectID  string
	client     *gpubsub.Client
	conn       *grpc.ClientConn
	ownsClient bool
}

// Config configures Google Pub/Sub transport construction.
type Config struct {
	ProjectID string
	URI       string
	Client    *gpubsub.Client
}

// New constructs a Google Pub/Sub-backed driver.
func New(ctx context.Context, cfg Config) (*Driver, error) {
	if cfg.Client != nil {
		if cfg.ProjectID == "" {
			return nil, errors.New("gcppubsubevents: ProjectID is required")
		}
		return &Driver{
			projectID: cfg.ProjectID,
			client:    cfg.Client,
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
	}, nil
}

// Driver reports the active backend kind.
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverGCPPubSub
}

// Ready checks Google Pub/Sub connectivity.
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
func (d *Driver) Close() error {
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
	pubTopic := d.client.Topic(topic)
	exists, err := pubTopic.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return pubTopic, nil
	}
	created, err := d.client.CreateTopic(ctx, topic)
	if err != nil {
		return nil, err
	}
	return created, nil
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
