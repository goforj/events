package kafkaevents

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/goforj/events/eventscore"
	"github.com/segmentio/kafka-go"
)

// Driver is a Kafka-backed events transport.
type Driver struct {
	brokers []string
	dialer  *kafka.Dialer
	writer  *kafka.Writer
}

// Config configures Kafka transport construction.
type Config struct {
	Brokers []string
	Dialer  *kafka.Dialer
	Writer  *kafka.Writer
}

// New constructs a Kafka-backed driver.
func New(cfg Config) (*Driver, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafkaevents: Brokers are required")
	}
	dialer := cfg.Dialer
	if dialer == nil {
		dialer = &kafka.Dialer{}
	}
	writer := cfg.Writer
	if writer == nil {
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireAll,
		}
	}
	return &Driver{
		brokers: append([]string(nil), cfg.Brokers...),
		dialer:  dialer,
		writer:  writer,
	}, nil
}

// Driver reports the active backend kind.
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverKafka
}

// Ready checks Kafka connectivity.
func (d *Driver) Ready(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	conn, err := d.dialer.DialContext(ctx, "tcp", d.brokers[0])
	if err != nil {
		return err
	}
	return conn.Close()
}

// PublishContext publishes a topic payload to Kafka.
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if err := d.ensureTopic(ctx, msg.Topic); err != nil {
		return err
	}
	return d.writer.WriteMessages(ctx, kafka.Message{
		Topic: msg.Topic,
		Value: msg.Payload,
	})
}

// SubscribeContext subscribes to a Kafka topic and forwards messages.
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := d.ensureTopic(ctx, topic); err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     d.brokers,
		Topic:       topic,
		Partition:   0,
		StartOffset: kafka.LastOffset,
		MaxAttempts: 3,
	})

	workerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			message, err := reader.ReadMessage(workerCtx)
			switch {
			case err == nil:
				_ = handler(context.Background(), eventscore.Message{
					Topic:   message.Topic,
					Payload: message.Value,
				})
			case errors.Is(err, context.Canceled), errors.Is(err, io.EOF), errors.Is(err, io.ErrClosedPipe):
				return
			default:
				if workerCtx.Err() != nil {
					return
				}
				return
			}
		}
	}()

	return subscription{
		cancel: cancel,
		done:   done,
		reader: reader,
	}, nil
}

// Close closes the underlying Kafka writer.
func (d *Driver) Close() error {
	return d.writer.Close()
}

func (d *Driver) ensureTopic(ctx context.Context, topic string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}

	conn, err := d.dialer.DialContext(ctx, "tcp", d.brokers[0])
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := d.dialer.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer func() { _ = controllerConn.Close() }()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		return err
	}
	return nil
}

type subscription struct {
	cancel context.CancelFunc
	done   <-chan struct{}
	reader *kafka.Reader
}

func (s subscription) Close() error {
	s.cancel()
	err := s.reader.Close()
	<-s.done
	return err
}
