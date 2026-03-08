package redisevents

import (
	"context"
	"errors"

	"github.com/goforj/events/eventscore"
	"github.com/redis/go-redis/v9"
)

// Driver is a Redis pub/sub-backed events transport.
type Driver struct {
	client *redis.Client
}

// Config configures Redis transport construction.
type Config struct {
	Addr   string
	Client *redis.Client
}

// New constructs a Redis pub/sub-backed driver.
func New(cfg Config) (*Driver, error) {
	if cfg.Client != nil {
		return &Driver{client: cfg.Client}, nil
	}
	if cfg.Addr == "" {
		return nil, errors.New("redisevents: Addr is required")
	}
	return &Driver{
		client: redis.NewClient(&redis.Options{Addr: cfg.Addr}),
	}, nil
}

// Driver reports the active backend kind.
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverRedis
}

// Ready checks Redis connectivity.
func (d *Driver) Ready(ctx context.Context) error {
	return d.client.Ping(ctx).Err()
}

// PublishContext publishes a topic payload via Redis pub/sub.
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	return d.client.Publish(ctx, msg.Topic, msg.Payload).Err()
}

// SubscribeContext subscribes to a Redis pub/sub channel.
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	pubsub := d.client.Subscribe(ctx, topic)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}
	ch := pubsub.Channel()
	workerCtx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		for {
			select {
			case <-workerCtx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				_ = handler(context.Background(), eventscore.Message{
					Topic:   msg.Channel,
					Payload: []byte(msg.Payload),
				})
			}
		}
	}()
	return subscription{pubsub: pubsub, cancel: cancel}, nil
}

// Close closes the underlying Redis client.
func (d *Driver) Close() error {
	return d.client.Close()
}

type subscription struct {
	pubsub *redis.PubSub
	cancel context.CancelFunc
}

func (s subscription) Close() error {
	s.cancel()
	return s.pubsub.Close()
}
