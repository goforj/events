package events

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/goforj/events/eventscore"
)

// Bus is the root event bus implementation.
// @group Bus
type Bus struct {
	*busState
	ctx context.Context
}

// busState holds the synchronization and resources shared by every context-bound bus handle.
type busState struct {
	driver    eventscore.Driver
	codec     Codec
	transport eventscore.DriverAPI
	mu        sync.RWMutex

	handlers      map[string][]registeredHandler
	transportSubs map[string]*transportTopic
}

// transportTopic tracks one transport subscription generation for a topic.
// A generation prevents callbacks from a closing subscription from reaching
// handlers registered against its replacement.
type transportTopic struct {
	generation   *transportGeneration
	setup        *transportSetup
	subscription eventscore.Subscription
}

// transportGeneration gives each transport subscription a stable identity.
type transportGeneration struct{}

// transportSetup lets concurrent subscribers share one transport handshake.
type transportSetup struct {
	done chan struct{}
	err  error
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
	if options.codec != nil && isNilInterface(options.codec) {
		return nil, ErrNilCodec
	}
	if options.codec == nil {
		options.codec = jsonCodec{}
	}
	if cfg.Transport != nil && isNilInterface(cfg.Transport) {
		return nil, ErrNilTransport
	}
	if cfg.Transport != nil {
		cfg.Driver = cfg.Transport.Driver()
	}
	if cfg.Driver == "" {
		cfg.Driver = eventscore.DriverSync
	}
	if cfg.Transport == nil && cfg.Driver != eventscore.DriverSync && cfg.Driver != eventscore.DriverNull {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedDriver, cfg.Driver)
	}
	return &Bus{
		busState: &busState{
			driver:        cfg.Driver,
			codec:         options.codec,
			transport:     cfg.Transport,
			handlers:      make(map[string][]registeredHandler),
			transportSubs: make(map[string]*transportTopic),
		},
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

// WithContext returns a derived bus handle bound to ctx for subsequent operations.
// @group Bus
func (b *Bus) WithContext(ctx context.Context) API {
	clone := *b
	clone.ctx = ctx
	return &clone
}

// Driver reports the active backend.
// @group Bus
//
// Example: inspect the active backend
//
//	bus, _ := events.NewSync()
//	fmt.Println(bus.Driver())
//	// Output: sync
func (b *Bus) Driver() eventscore.Driver {
	return b.driver
}

// context returns the handle context while normalizing nil contexts at the API boundary.
func (b *Bus) context() context.Context {
	if b.ctx == nil {
		return context.Background()
	}
	return b.ctx
}

// Ready reports whether the bus is ready.
// @group Bus
//
// Example: check readiness
//
//	bus, _ := events.NewSync()
//	fmt.Println(bus.Ready() == nil)
//	// Output: true
func (b *Bus) Ready() error {
	return b.ready(b.context())
}

// ready performs the context-aware readiness check used by derived handles.
func (b *Bus) ready(ctx context.Context) error {
	if b.transport != nil {
		return b.transport.Ready(ctx)
	}
	return nil
}

// Publish publishes an event using the background context.
// @group Publish
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
	return b.publish(b.context(), event)
}

// publish performs the context-aware publish used by derived handles.
func (b *Bus) publish(ctx context.Context, event any) error {
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

// dispatchMessage delivers a local message to the current topic handlers.
func (b *Bus) dispatchMessage(ctx context.Context, msg eventscore.Message) error {
	b.mu.RLock()
	handlers := append([]registeredHandler(nil), b.handlers[msg.Topic]...)
	b.mu.RUnlock()
	return b.dispatchHandlers(ctx, msg, handlers)
}

// dispatchTransportMessage ignores callbacks from starting or retired generations.
func (b *Bus) dispatchTransportMessage(ctx context.Context, msg eventscore.Message, generation *transportGeneration) error {
	b.mu.RLock()
	topic := b.transportSubs[msg.Topic]
	if topic == nil || topic.generation != generation || topic.setup != nil {
		b.mu.RUnlock()
		return nil
	}
	handlers := make([]registeredHandler, 0, len(b.handlers[msg.Topic]))
	for _, handler := range b.handlers[msg.Topic] {
		if handler.generation == generation {
			handlers = append(handlers, handler)
		}
	}
	b.mu.RUnlock()
	return b.dispatchHandlers(ctx, msg, handlers)
}

// dispatchHandlers decodes separately for each handler so pointer and value signatures stay independent.
func (b *Bus) dispatchHandlers(ctx context.Context, msg eventscore.Message, handlers []registeredHandler) error {
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
// @group Subscribe
//
// Example: subscribe to a typed event
//
//	type UserCreated struct {
//		ID string `json:"id"`
//	}
//
//	bus, _ := events.NewSync()
//	sub, _ := bus.Subscribe(func(ctx context.Context, event UserCreated) error {
//		fmt.Println(event.ID)
//		return nil
//	})
//	defer sub.Close()
//	_ = bus.Publish(UserCreated{ID: "123"})
//	// Output: 123
func (b *Bus) Subscribe(handler any) (Subscription, error) {
	return b.subscribe(b.context(), handler)
}

// subscribe performs the context-aware registration used by derived handles.
func (b *Bus) subscribe(ctx context.Context, handler any) (Subscription, error) {
	registered, err := newRegisteredHandler(handler)
	if err != nil {
		return nil, err
	}
	if b.driver == eventscore.DriverNull {
		return noopSubscription{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if b.transport == nil {
		b.mu.Lock()
		b.handlers[registered.topic] = append(b.handlers[registered.topic], registered)
		b.mu.Unlock()
		return &subscription{bus: b, topic: registered.topic, id: registered.id}, nil
	}

	b.mu.Lock()
	topic := b.transportSubs[registered.topic]
	if topic == nil {
		topic = &transportTopic{
			generation: &transportGeneration{},
			setup:      &transportSetup{done: make(chan struct{})},
		}
		b.transportSubs[registered.topic] = topic
	}
	registered.generation = topic.generation
	b.handlers[registered.topic] = append(b.handlers[registered.topic], registered)
	setup := topic.setup
	if setup == nil {
		b.mu.Unlock()
		return &subscription{bus: b, topic: registered.topic, id: registered.id, generation: registered.generation}, nil
	}
	owner := len(b.handlers[registered.topic]) == 1
	b.mu.Unlock()
	if !owner {
		return b.waitForTransportSetup(ctx, registered, setup)
	}

	transportSub, err := b.transport.SubscribeContext(ctx, registered.topic, func(msgCtx context.Context, msg eventscore.Message) error {
		return b.dispatchTransportMessage(msgCtx, msg, registered.generation)
	})
	if err == nil && isNilInterface(transportSub) {
		err = ErrNilSubscription
	}

	b.mu.Lock()
	if err != nil {
		b.removeGenerationLocked(registered.topic, registered.generation)
		delete(b.transportSubs, registered.topic)
		setup.err = err
		close(setup.done)
		b.mu.Unlock()
		return nil, err
	}
	topic.subscription = transportSub
	topic.setup = nil
	close(setup.done)
	b.mu.Unlock()
	return &subscription{bus: b, topic: registered.topic, id: registered.id, generation: registered.generation}, nil
}

// waitForTransportSetup waits for the first subscriber's shared transport handshake.
func (b *Bus) waitForTransportSetup(ctx context.Context, registered registeredHandler, setup *transportSetup) (Subscription, error) {
	select {
	case <-setup.done:
		if setup.err != nil {
			return nil, setup.err
		}
		return &subscription{bus: b, topic: registered.topic, id: registered.id, generation: registered.generation}, nil
	case <-ctx.Done():
		cleanupErr := b.removeHandler(registered.topic, registered.id, registered.generation)
		if cleanupErr == nil {
			return nil, ctx.Err()
		}
		return nil, errors.Join(ctx.Err(), cleanupErr)
	}
}

// removeGenerationLocked rolls back every handler that joined a failed setup.
func (b *Bus) removeGenerationLocked(topic string, generation *transportGeneration) {
	handlers := b.handlers[topic]
	kept := handlers[:0]
	for _, handler := range handlers {
		if handler.generation != generation {
			kept = append(kept, handler)
		}
	}
	if len(kept) == 0 {
		delete(b.handlers, topic)
		return
	}
	b.handlers[topic] = kept
}

type noopSubscription struct{}

// Close preserves the subscription lifecycle contract for the drop-only bus.
func (noopSubscription) Close() error { return nil }

type subscription struct {
	bus        *Bus
	topic      string
	id         uint64
	generation *transportGeneration
	once       sync.Once
	err        error
}

// Close removes the handler and releases the shared transport subscription when it is last.
func (s *subscription) Close() error {
	s.once.Do(func() {
		s.err = s.bus.removeHandler(s.topic, s.id, s.generation)
	})
	return s.err
}

// removeHandler detaches state under the bus lock and closes external resources after unlocking.
func (b *Bus) removeHandler(topic string, id uint64, generation *transportGeneration) error {
	b.mu.Lock()
	handlers := b.handlers[topic]
	removed := false
	for i := range handlers {
		if handlers[i].id == id {
			handlers = append(handlers[:i], handlers[i+1:]...)
			removed = true
			break
		}
	}
	if !removed {
		b.mu.Unlock()
		return nil
	}
	if len(handlers) == 0 {
		delete(b.handlers, topic)
	} else {
		b.handlers[topic] = handlers
	}

	var transportSub eventscore.Subscription
	remainingGeneration := false
	for _, handler := range handlers {
		if handler.generation == generation {
			remainingGeneration = true
			break
		}
	}
	transportTopic := b.transportSubs[topic]
	if !remainingGeneration && transportTopic != nil && transportTopic.generation == generation {
		transportSub = transportTopic.subscription
		delete(b.transportSubs, topic)
	}
	b.mu.Unlock()

	if transportSub != nil {
		return transportSub.Close()
	}
	return nil
}

// buildMessage resolves routing before serialization so invalid events never reach a codec.
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
