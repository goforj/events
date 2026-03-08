package events

import (
	"context"
	"sync"

	"github.com/goforj/events/eventscore"
)

// Bus is the root event bus implementation.
// @group Core
//
// Example: keep a concrete bus reference
//
//	bus, _ := events.NewSync()
//	var root *events.Bus = bus
//	fmt.Println(root.Driver())
//	// Output: sync
type Bus struct {
	driver    eventscore.Driver
	codec     Codec
	transport eventscore.DriverAPI
	mu        sync.RWMutex

	handlers      map[string][]registeredHandler
	transportSubs map[string]eventscore.Subscription
}

// New constructs a root bus for the requested driver.
// @group Construction
//
// Example: construct a bus from config
//
//	bus, _ := events.New(events.Config{Driver: "sync"})
//	fmt.Println(bus.Driver())
//	// Output: sync
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
// @group Construction
//
// Example: construct a sync bus
//
//	bus, _ := events.NewSync()
//	fmt.Println(bus.Driver())
//	// Output: sync
func NewSync(opts ...Option) (*Bus, error) {
	return New(Config{Driver: eventscore.DriverSync}, opts...)
}

// NewNull constructs the root null bus.
// @group Construction
//
// Example: construct a null bus
//
//	bus, _ := events.NewNull()
//	fmt.Println(bus.Driver())
//	// Output: null
func NewNull(opts ...Option) (*Bus, error) {
	return New(Config{Driver: eventscore.DriverNull}, opts...)
}

// Driver reports the active backend.
// @group Core
//
// Example: inspect the active backend
//
//	bus, _ := events.NewSync()
//	fmt.Println(bus.Driver())
//	// Output: sync
func (b *Bus) Driver() eventscore.Driver {
	return b.driver
}

// Ready reports whether the bus is ready.
// @group Core
//
// Example: check readiness
//
//	bus, _ := events.NewSync()
//	fmt.Println(bus.Ready() == nil)
//	// Output: true
func (b *Bus) Ready() error {
	return b.ReadyContext(context.Background())
}

// ReadyContext reports whether the bus is ready.
// @group Core
//
// Example: check readiness with a caller context
//
//	bus, _ := events.NewSync()
//	fmt.Println(bus.ReadyContext(context.Background()) == nil)
//	// Output: true
func (b *Bus) ReadyContext(ctx context.Context) error {
	if b.transport != nil {
		return b.transport.Ready(ctx)
	}
	return nil
}

// Publish publishes an event using the background context.
// @group Core
//
// Example: publish a typed event
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	_, _ = bus.Subscribe(func(event UserCreated) {
//		fmt.Println(event.ID)
//	})
//	_ = bus.Publish(UserCreated{ID: "123"})
//	// Output: 123
func (b *Bus) Publish(event any) error {
	return b.PublishContext(context.Background(), event)
}

// PublishContext publishes an event using the configured codec and dispatch flow.
// @group Core
//
// Example: publish with a caller context
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	_, _ = bus.Subscribe(func(ctx context.Context, event UserCreated) error {
//		fmt.Println(event.ID, ctx != nil)
//		return nil
//	})
//	_ = bus.PublishContext(context.Background(), UserCreated{ID: "123"})
//	// Output: 123 true
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
// @group Core
//
// Example: subscribe to a typed event
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	sub, _ := bus.Subscribe(func(ctx context.Context, event UserCreated) error {
//		_ = ctx
//		_ = event
//		return nil
//	})
//	defer sub.Close()
func (b *Bus) Subscribe(handler any) (Subscription, error) {
	return b.subscribeContext(context.Background(), handler)
}

// SubscribeContext registers a typed handler.
// @group Core
//
// Example: subscribe with a caller context
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	sub, _ := bus.SubscribeContext(context.Background(), func(ctx context.Context, event UserCreated) error {
//		_ = ctx
//		_ = event
//		return nil
//	})
//	defer sub.Close()
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
