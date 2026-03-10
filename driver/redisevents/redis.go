package redisevents

import (
	"context"
	"errors"

	"github.com/goforj/events/eventscore"
	"github.com/redis/go-redis/v9"
)

// Driver is a Redis pub/sub-backed events transport.
// @group Drivers
//
// Example: keep a Redis driver reference
//
//	var driver *redisevents.Driver
//	fmt.Println(driver == nil)
//	// Output: true
type Driver struct {
	client *redis.Client
}

// Config configures Redis transport construction.
// @group Driver Config
//
// Example: define Redis driver config
//
//	cfg := redisevents.Config{Addr: "127.0.0.1:6379"}
//	_ = cfg
//
// Example: define Redis driver config with all fields
//
//	cfg := redisevents.Config{
//		Addr:   "127.0.0.1:6379",
//		Client: nil, // default: nil constructs a client from Addr
//	}
//	_ = cfg
type Config struct {
	Addr   string
	Client *redis.Client
}

// New constructs a Redis pub/sub-backed driver.
// @group Driver Constructors
//
// Example: construct a Redis driver
//
//	driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
//	_ = driver
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
// @group Drivers
//
// Example: inspect the driver kind
//
//	driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
//	_ = driver
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverRedis
}

// Ready checks Redis connectivity.
// @group Drivers
//
// Example: check Redis connectivity
//
//	driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
//	fmt.Println(driver.Ready(context.Background()) == nil)
//	// Output: true
func (d *Driver) Ready(ctx context.Context) error {
	return d.client.Ping(ctx).Err()
}

// PublishContext publishes a topic payload via Redis pub/sub.
// @group Drivers
//
// Example: publish a raw message through Redis
//
//	driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
//	_ = driver.PublishContext(context.Background(), eventscore.Message{
//		Topic:   "users.created",
//		Payload: []byte(`{"id":"123"}`),
//	})
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	return d.client.Publish(ctx, msg.Topic, msg.Payload).Err()
}

// SubscribeContext subscribes to a Redis pub/sub channel.
// @group Drivers
//
// Example: subscribe to a Redis channel
//
//	driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
//	sub, _ := driver.SubscribeContext(context.Background(), "users.created", func(ctx context.Context, msg eventscore.Message) error {
//		_ = ctx
//		_ = msg
//		return nil
//	})
//	fmt.Println(sub != nil)
//	// Output: true
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
// @group Drivers
//
// Example: close a Redis driver
//
//	driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
//	_ = driver.Close()
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
