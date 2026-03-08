package kafkaevents

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/goforj/events/eventscore"
	"github.com/segmentio/kafka-go"
)

// Driver is a Kafka-backed events transport.
//
// Example: keep a Kafka driver reference
//
//	var driver *kafkaevents.Driver
//	fmt.Println(driver == nil)
//	// Output: true
type Driver struct {
	brokers []string
	dialer  *kafka.Dialer
	writer  *kafka.Writer

	ensuredMu     sync.RWMutex
	ensuredTopics map[string]struct{}
}

// Config configures Kafka transport construction.
//
// Example: define Kafka driver config
//
//	cfg := kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}}
//	fmt.Println(cfg.Brokers[0])
//	// Output: 127.0.0.1:9092
type Config struct {
	Brokers []string
	Dialer  *kafka.Dialer
	Writer  *kafka.Writer
}

const (
	defaultReaderMaxWait      = 100 * time.Millisecond
	defaultReaderBatchTimeout = 100 * time.Millisecond
	defaultWriterBatchTimeout = 10 * time.Millisecond
)

// New constructs a Kafka-backed driver.
//
// Example: construct a Kafka driver
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	fmt.Println(driver != nil)
//	// Output: true
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
			BatchSize:              1,
			BatchTimeout:           defaultWriterBatchTimeout,
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireAll,
		}
	}
	return &Driver{
		brokers:       append([]string(nil), cfg.Brokers...),
		dialer:        dialer,
		writer:        writer,
		ensuredTopics: make(map[string]struct{}),
	}, nil
}

// Driver reports the active backend kind.
//
// Example: inspect the driver kind
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	fmt.Println(driver.Driver())
//	// Output: kafka
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverKafka
}

// Ready checks Kafka connectivity.
//
// Example: check Kafka connectivity
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	fmt.Println(driver.Ready(context.Background()) == nil)
//	// Output: true
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
//
// Example: publish a raw message through Kafka
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	_ = driver.PublishContext(context.Background(), eventscore.Message{
//		Topic:   "users.created",
//		Payload: []byte(`{"id":"123"}`),
//	})
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
//
// Example: subscribe to a Kafka topic
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	sub, _ := driver.SubscribeContext(context.Background(), "users.created", func(ctx context.Context, msg eventscore.Message) error {
//		_ = ctx
//		_ = msg
//		return nil
//	})
//	fmt.Println(sub != nil)
//	// Output: true
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := d.ensureTopic(ctx, topic); err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          d.brokers,
		Topic:            topic,
		Partition:        0,
		StartOffset:      kafka.LastOffset,
		MaxWait:          defaultReaderMaxWait,
		ReadBatchTimeout: defaultReaderBatchTimeout,
		MaxAttempts:      3,
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
//
// Example: close a Kafka driver
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	fmt.Println(driver.Close() == nil)
//	// Output: true
func (d *Driver) Close() error {
	return d.writer.Close()
}

func (d *Driver) ensureTopic(ctx context.Context, topic string) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if d.topicEnsured(topic) {
		return nil
	}

	d.ensuredMu.Lock()
	defer d.ensuredMu.Unlock()
	if _, ok := d.ensuredTopics[topic]; ok {
		return nil
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
	d.ensuredTopics[topic] = struct{}{}
	return nil
}

func (d *Driver) topicEnsured(topic string) bool {
	d.ensuredMu.RLock()
	defer d.ensuredMu.RUnlock()
	_, ok := d.ensuredTopics[topic]
	return ok
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
