package natsevents

import (
	"context"
	"errors"

	"github.com/goforj/events/eventscore"
	"github.com/nats-io/nats.go"
)

// Driver is a NATS-backed events transport.
type Driver struct {
	conn *nats.Conn
}

// Config configures NATS transport construction.
type Config struct {
	URL  string
	Conn *nats.Conn
}

// New connects a NATS-backed driver from config.
func New(cfg Config) (*Driver, error) {
	if cfg.Conn != nil {
		return &Driver{conn: cfg.Conn}, nil
	}
	if cfg.URL == "" {
		return nil, errors.New("natsevents: URL is required")
	}
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, err
	}
	return &Driver{conn: conn}, nil
}

// Driver reports the active backend kind.
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverNATS
}

// Ready checks that the NATS connection is healthy.
func (d *Driver) Ready(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return d.conn.Flush()
}

// PublishContext publishes a topic payload to NATS.
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if err := d.conn.Publish(msg.Topic, msg.Payload); err != nil {
		return err
	}
	return d.conn.Flush()
}

// SubscribeContext subscribes to a NATS subject and forwards messages.
func (d *Driver) SubscribeContext(_ context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	sub, err := d.conn.Subscribe(topic, func(msg *nats.Msg) {
		_ = handler(context.Background(), eventscore.Message{
			Topic:   msg.Subject,
			Payload: msg.Data,
		})
	})
	if err != nil {
		return nil, err
	}
	if err := d.conn.Flush(); err != nil {
		_ = sub.Unsubscribe()
		return nil, err
	}
	return subscription{sub: sub}, nil
}

// Close drains the underlying NATS connection.
func (d *Driver) Close() error {
	d.conn.Close()
	return nil
}

type subscription struct {
	sub *nats.Subscription
}

func (s subscription) Close() error {
	return s.sub.Unsubscribe()
}
