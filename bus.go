package events

import (
	"context"
	"sync"

	"github.com/goforj/events/eventscore"
)

// Bus is the root event bus implementation.
type Bus struct {
	driver    eventscore.Driver
	codec     Codec
	transport eventscore.DriverAPI
	mu        sync.RWMutex

	handlers      map[string][]registeredHandler
	transportSubs map[string]eventscore.Subscription
}

// New constructs a root bus for the requested driver.
func New(cfg Config, opts ...Option) (*Bus, error) {
	options := options{codec: cfg.Codec}
	options.apply(opts)
	if options.codec == nil {
		options.codec = jsonCodec{}
	}
	if cfg.Transport != nil {
		cfg.Driver = cfg.Transport.Driver()
	}
	if cfg.Driver == "" {
		cfg.Driver = eventscore.DriverSync
	}
	return &Bus{
		driver:        cfg.Driver,
		codec:         options.codec,
		transport:     cfg.Transport,
		handlers:      make(map[string][]registeredHandler),
		transportSubs: make(map[string]eventscore.Subscription),
	}, nil
}

// NewSync constructs the root sync bus.
func NewSync(opts ...Option) (*Bus, error) {
	return New(Config{Driver: eventscore.DriverSync}, opts...)
}

// NewNull constructs the root null bus.
func NewNull(opts ...Option) (*Bus, error) {
	return New(Config{Driver: eventscore.DriverNull}, opts...)
}

// Driver reports the active backend.
func (b *Bus) Driver() eventscore.Driver {
	return b.driver
}

// Ready reports whether the bus is ready.
func (b *Bus) Ready() error {
	return b.ReadyContext(context.Background())
}

// ReadyContext reports whether the bus is ready.
func (b *Bus) ReadyContext(ctx context.Context) error {
	if b.transport != nil {
		return b.transport.Ready(ctx)
	}
	return nil
}

// Publish publishes an event using the background context.
func (b *Bus) Publish(event any) error {
	return b.PublishContext(context.Background(), event)
}

// PublishContext publishes an event using the configured codec and dispatch flow.
func (b *Bus) PublishContext(ctx context.Context, event any) error {
	if b.driver == eventscore.DriverNull {
		return nil
	}
	msg, err := b.buildMessage(event)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if b.transport != nil {
		return b.transport.PublishContext(ctx, msg)
	}
	return b.dispatchMessage(ctx, msg)
}

func (b *Bus) dispatchMessage(ctx context.Context, msg eventscore.Message) error {

	b.mu.RLock()
	handlers := append([]registeredHandler(nil), b.handlers[msg.Topic]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		value, err := handler.decodePayload(b.codec, msg.Payload)
		if err != nil {
			return err
		}
		if err := handler.invoke(ctx, value); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler using the background context.
func (b *Bus) Subscribe(handler any) (Subscription, error) {
	return b.subscribeContext(context.Background(), handler)
}

// SubscribeContext registers a typed handler.
func (b *Bus) SubscribeContext(ctx context.Context, handler any) (Subscription, error) {
	return b.subscribeContext(ctx, handler)
}

func (b *Bus) subscribeContext(ctx context.Context, handler any) (Subscription, error) {
	registered, err := newRegisteredHandler(handler)
	if err != nil {
		return nil, err
	}
	if b.driver == eventscore.DriverNull {
		return noopSubscription{}, nil
	}
	b.mu.Lock()
	first := len(b.handlers[registered.topic]) == 0
	b.handlers[registered.topic] = append(b.handlers[registered.topic], registered)
	b.mu.Unlock()
	if b.transport != nil && first {
		transportSub, err := b.transport.SubscribeContext(ctx, registered.topic, func(msgCtx context.Context, msg eventscore.Message) error {
			return b.dispatchMessage(msgCtx, msg)
		})
		if err != nil {
			b.mu.Lock()
			handlers := b.handlers[registered.topic]
			if len(handlers) > 0 {
				b.handlers[registered.topic] = handlers[:len(handlers)-1]
				if len(b.handlers[registered.topic]) == 0 {
					delete(b.handlers, registered.topic)
				}
			}
			b.mu.Unlock()
			return nil, err
		}
		b.mu.Lock()
		b.transportSubs[registered.topic] = transportSub
		b.mu.Unlock()
	}
	return &subscription{bus: b, topic: registered.topic, id: registered.id}, nil
}

type noopSubscription struct{}

func (noopSubscription) Close() error { return nil }

type subscription struct {
	bus   *Bus
	topic string
	id    uint64
	once  sync.Once
}

func (s *subscription) Close() error {
	s.once.Do(func() {
		s.bus.mu.Lock()
		defer s.bus.mu.Unlock()
		handlers := s.bus.handlers[s.topic]
		for i := range handlers {
			if handlers[i].id == s.id {
				s.bus.handlers[s.topic] = append(handlers[:i], handlers[i+1:]...)
				if len(s.bus.handlers[s.topic]) == 0 {
					if sub := s.bus.transportSubs[s.topic]; sub != nil {
						_ = sub.Close()
						delete(s.bus.transportSubs, s.topic)
					}
					delete(s.bus.handlers, s.topic)
				}
				return
			}
		}
	})
	return nil
}

func (b *Bus) buildMessage(event any) (eventscore.Message, error) {
	topic, _, err := resolveTopic(event)
	if err != nil {
		return eventscore.Message{}, err
	}
	payload, err := b.codec.Marshal(event)
	if err != nil {
		return eventscore.Message{}, err
	}
	return eventscore.Message{
		Topic:   topic,
		Payload: payload,
	}, nil
}
