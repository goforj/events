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
// @group Drivers
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
// @group Config
//
// Example: define Kafka driver config
//
//	cfg := kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}}
//	_ = cfg
//
// Example: define Kafka driver config with all fields
//
//	cfg := kafkaevents.Config{
//		Brokers: []string{"127.0.0.1:9092"},
//		Dialer:  nil, // default: nil uses a zero-value kafka.Dialer
//		Writer:  nil, // default: nil builds a writer with single-message, auto-topic defaults
//	}
//	_ = cfg
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
// @group Driver Constructors
//
// Example: construct a Kafka driver
//
//	driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
//	_ = driver
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
// @group Drivers
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverKafka
}

// Ready checks Kafka connectivity.
// @group Drivers
func (d *Driver) Ready(ctx context.Context) error {
	ctx = normalizeContext(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	conn, err := d.dialer.DialContext(ctx, "tcp", d.brokers[0])
	if err != nil {
		return err
	}
	return conn.Close()
}

// PublishContext publishes a topic payload to Kafka.
// @group Drivers
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	ctx = normalizeContext(ctx)
	if ctx.Err() != nil {
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
// @group Drivers
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	ctx = normalizeContext(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if handler == nil {
		return nil, errors.New("kafkaevents: handler is required")
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

	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			message, err := reader.ReadMessage(workerCtx)
			switch {
			case err == nil:
				_ = handler(workerCtx, eventscore.Message{
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

	return &subscription{
		cancel: cancel,
		done:   done,
		reader: reader,
	}, nil
}

// normalizeContext keeps direct driver calls consistent with the root bus facade.
func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// Close closes the underlying Kafka writer.
// @group Lifecycle
func (d *Driver) Close() error {
	return d.writer.Close()
}

// ensureTopic creates each single-partition compatibility topic at most once per driver.
func (d *Driver) ensureTopic(ctx context.Context, topic string) error {
	ctx = normalizeContext(ctx)
	if ctx.Err() != nil {
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

// topicEnsured keeps the common publish path free of the creation mutex after setup.
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
	once   sync.Once
	err    error
}

// Close cancels delivery once and keeps the first teardown result for repeated calls.
func (s *subscription) Close() error {
	s.once.Do(func() {
		s.cancel()
		s.err = s.reader.Close()
		<-s.done
	})
	return s.err
}
