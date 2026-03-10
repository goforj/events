package natsjetstreamevents

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goforj/events/eventscore"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var nextConsumerID atomic.Uint64

var nameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

const (
	defaultSubjectPrefix     = "events."
	defaultStreamNamePrefix  = "EVENTS_"
	defaultInactiveThreshold = 30 * time.Second
	defaultAckWait           = 30 * time.Second
	defaultFetchMaxWait      = 250 * time.Millisecond
	defaultStorage           = jetstream.MemoryStorage
	maxNameLen               = 80
)

// Driver is a NATS JetStream-backed events transport.
// @group Drivers
//
// Example: keep a NATS JetStream driver reference
//
//	var driver *natsjetstreamevents.Driver
//	fmt.Println(driver == nil)
//	// Output: true
type Driver struct {
	conn *nats.Conn
	js   jetstream.JetStream

	subjectPrefix     string
	streamNamePrefix  string
	inactiveThreshold time.Duration
	ackWait           time.Duration
	fetchMaxWait      time.Duration
	storage           jetstream.StorageType

	streamsMu sync.RWMutex
	streams   map[string]string
}

// Config configures NATS JetStream transport construction.
// @group Config
//
// Example: define NATS JetStream driver config
//
//	cfg := natsjetstreamevents.Config{URL: "nats://127.0.0.1:4222"}
//	_ = cfg
//
// Example: define NATS JetStream driver config with all fields
//
//	cfg := natsjetstreamevents.Config{
//		URL:               "nats://127.0.0.1:4222",
//		Conn:              nil,                    // default: nil dials URL instead of reusing an existing connection
//		SubjectPrefix:     "events.",              // default: "events."
//		StreamNamePrefix:  "EVENTS_",              // default: "EVENTS_"
//		InactiveThreshold: 30 * time.Second,       // default: 30s
//		AckWait:           30 * time.Second,       // default: 30s
//		FetchMaxWait:      250 * time.Millisecond, // default: 250ms
//		Storage:           jetstream.MemoryStorage,// default: MemoryStorage
//	}
//	_ = cfg
type Config struct {
	URL               string
	Conn              *nats.Conn
	SubjectPrefix     string
	StreamNamePrefix  string
	InactiveThreshold time.Duration
	AckWait           time.Duration
	FetchMaxWait      time.Duration
	Storage           jetstream.StorageType
}

// New connects a NATS JetStream-backed driver from config.
// @group Driver Constructors
//
// Example: construct a NATS JetStream driver
//
//	driver, _ := natsjetstreamevents.New(natsjetstreamevents.Config{URL: "nats://127.0.0.1:4222"})
//	_ = driver
func New(cfg Config) (*Driver, error) {
	conn := cfg.Conn
	if conn == nil {
		if cfg.URL == "" {
			return nil, errors.New("natsjetstreamevents: URL is required")
		}
		var err error
		conn, err = nats.Connect(cfg.URL)
		if err != nil {
			return nil, err
		}
	}
	js, err := jetstream.New(conn)
	if err != nil {
		if cfg.Conn == nil {
			conn.Close()
		}
		return nil, err
	}
	subjectPrefix := cfg.SubjectPrefix
	if subjectPrefix == "" {
		subjectPrefix = defaultSubjectPrefix
	}
	streamNamePrefix := cfg.StreamNamePrefix
	if streamNamePrefix == "" {
		streamNamePrefix = defaultStreamNamePrefix
	}
	inactiveThreshold := cfg.InactiveThreshold
	if inactiveThreshold <= 0 {
		inactiveThreshold = defaultInactiveThreshold
	}
	ackWait := cfg.AckWait
	if ackWait <= 0 {
		ackWait = defaultAckWait
	}
	fetchMaxWait := cfg.FetchMaxWait
	if fetchMaxWait <= 0 {
		fetchMaxWait = defaultFetchMaxWait
	}
	storage := cfg.Storage
	if storage == 0 {
		storage = defaultStorage
	}
	return &Driver{
		conn:              conn,
		js:                js,
		subjectPrefix:     subjectPrefix,
		streamNamePrefix:  streamNamePrefix,
		inactiveThreshold: inactiveThreshold,
		ackWait:           ackWait,
		fetchMaxWait:      fetchMaxWait,
		storage:           storage,
		streams:           make(map[string]string),
	}, nil
}

// Driver reports the active backend kind.
// @group Drivers
func (d *Driver) Driver() eventscore.Driver {
	return eventscore.DriverNATSJetStream
}

// Ready checks JetStream connectivity.
// @group Drivers
func (d *Driver) Ready(ctx context.Context) error {
	ctx = normalizeContext(ctx)
	_, err := d.js.AccountInfo(ctx)
	return err
}

// PublishContext publishes a topic payload through JetStream.
// @group Drivers
func (d *Driver) PublishContext(ctx context.Context, msg eventscore.Message) error {
	ctx = normalizeContext(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	subject := d.subject(msg.Topic)
	streamName, err := d.ensureStream(ctx, msg.Topic, subject)
	if err != nil {
		return err
	}
	_, err = d.js.Publish(ctx, subject, msg.Payload, jetstream.WithExpectStream(streamName))
	return err
}

// SubscribeContext subscribes to a topic through an ephemeral pull consumer.
// @group Drivers
func (d *Driver) SubscribeContext(ctx context.Context, topic string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	ctx = normalizeContext(ctx)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	subject := d.subject(topic)
	streamName, err := d.ensureStream(ctx, topic, subject)
	if err != nil {
		return nil, err
	}
	consumerName := d.consumerName(topic)
	consumer, err := d.js.CreateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:              consumerName,
		FilterSubject:     subject,
		DeliverPolicy:     jetstream.DeliverNewPolicy,
		AckPolicy:         jetstream.AckExplicitPolicy,
		AckWait:           d.ackWait,
		InactiveThreshold: d.inactiveThreshold,
		MemoryStorage:     d.storage == jetstream.MemoryStorage,
	})
	if err != nil {
		return nil, err
	}

	workerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.consume(workerCtx, topic, consumer, handler)
	}()

	return subscription{
		cancel:       cancel,
		done:         done,
		driver:       d,
		streamName:   streamName,
		consumerName: consumerName,
	}, nil
}

// Close closes the underlying NATS connection.
// @group Lifecycle
func (d *Driver) Close() error {
	d.conn.Close()
	return nil
}

func (d *Driver) consume(ctx context.Context, topic string, consumer jetstream.Consumer, handler eventscore.MessageHandler) {
	messages, err := consumer.Messages()
	if err != nil {
		return
	}
	defer messages.Stop()

	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := messages.Next(jetstream.NextMaxWait(d.fetchMaxWait))
		switch {
		case err == nil:
			_ = handler(context.Background(), eventscore.Message{
				Topic:   topic,
				Payload: msg.Data(),
			})
			_ = msg.Ack()
		case errors.Is(err, context.Canceled), errors.Is(err, nats.ErrConnectionClosed):
			return
		default:
			if ctx.Err() != nil {
				return
			}
		}
	}
}

func (d *Driver) ensureStream(ctx context.Context, topic, subject string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if name, ok := d.cachedStream(topic); ok {
		return name, nil
	}

	d.streamsMu.Lock()
	defer d.streamsMu.Unlock()
	if name, ok := d.streams[topic]; ok {
		return name, nil
	}
	streamName := d.streamName(topic)
	_, err := d.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subject},
		Retention: jetstream.InterestPolicy,
		Storage:   d.storage,
		MaxAge:    d.ackWait * 4,
	})
	if err != nil {
		return "", err
	}
	d.streams[topic] = streamName
	return streamName, nil
}

func (d *Driver) cachedStream(topic string) (string, bool) {
	d.streamsMu.RLock()
	defer d.streamsMu.RUnlock()
	name, ok := d.streams[topic]
	return name, ok
}

func (d *Driver) subject(topic string) string {
	return d.subjectPrefix + topic
}

func (d *Driver) streamName(topic string) string {
	return sanitizeName(d.streamNamePrefix+topic, maxNameLen)
}

func (d *Driver) consumerName(topic string) string {
	return sanitizeName(fmt.Sprintf("C_%s_%d", topic, nextConsumerID.Add(1)), maxNameLen)
}

func sanitizeName(value string, maxLen int) string {
	value = nameSanitizer.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		value = "events"
	}
	if len(value) > maxLen {
		value = value[:maxLen]
	}
	return value
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

type subscription struct {
	cancel       context.CancelFunc
	done         <-chan struct{}
	driver       *Driver
	streamName   string
	consumerName string
}

func (s subscription) Close() error {
	s.cancel()
	<-s.done
	return s.driver.js.DeleteConsumer(context.Background(), s.streamName, s.consumerName)
}
