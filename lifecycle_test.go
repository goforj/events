package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/goforj/events/eventscore"
)

const lifecycleTestDeadline = 2 * time.Second

type lifecycleTransport struct {
	mu sync.Mutex

	handlers        []eventscore.MessageHandler
	subscriptions   []*lifecycleTransportSubscription
	subscribeCalls  int
	subscribeErr    error
	subscribeStart  chan struct{}
	subscribeGate   chan struct{}
	nilSubscription bool
	closeErr        error
}

// Driver reports the deterministic backend used by the lifecycle fixture.
func (t *lifecycleTransport) Driver() eventscore.Driver {
	return eventscore.DriverNATS
}

// Ready reports that the deterministic lifecycle fixture is ready.
func (t *lifecycleTransport) Ready(context.Context) error {
	return nil
}

// PublishContext delivers through the currently active fixture callbacks.
func (t *lifecycleTransport) PublishContext(ctx context.Context, msg eventscore.Message) error {
	t.mu.Lock()
	handlers := append([]eventscore.MessageHandler(nil), t.handlers...)
	subs := append([]*lifecycleTransportSubscription(nil), t.subscriptions...)
	t.mu.Unlock()
	for i, handler := range handlers {
		subs[i].inflight.Add(1)
		err := handler(ctx, msg)
		subs[i].inflight.Done()
		if err != nil {
			return err
		}
	}
	return nil
}

// SubscribeContext records one callback and can pause or fail the setup handshake deterministically.
func (t *lifecycleTransport) SubscribeContext(_ context.Context, _ string, handler eventscore.MessageHandler) (eventscore.Subscription, error) {
	t.mu.Lock()
	t.subscribeCalls++
	start := t.subscribeStart
	gate := t.subscribeGate
	err := t.subscribeErr
	nilSubscription := t.nilSubscription
	t.mu.Unlock()
	if start != nil {
		select {
		case <-start:
		default:
			close(start)
		}
	}
	if gate != nil {
		<-gate
	}
	if err != nil {
		return nil, err
	}
	if nilSubscription {
		var sub *lifecycleTransportSubscription
		return sub, nil
	}

	sub := &lifecycleTransportSubscription{transport: t, closeErr: t.closeErr, closeStarted: make(chan struct{})}
	t.mu.Lock()
	t.handlers = append(t.handlers, handler)
	t.subscriptions = append(t.subscriptions, sub)
	t.mu.Unlock()
	return sub, nil
}

// TestTransportSubscribeRejectsTypedNilSubscription verifies invalid driver output fails during registration.
func TestTransportSubscribeRejectsTypedNilSubscription(t *testing.T) {
	transport := &lifecycleTransport{nilSubscription: true}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := bus.Subscribe(func(userCreated) {}); !errors.Is(err, ErrNilSubscription) {
		t.Fatalf("Subscribe error = %v, want %v", err, ErrNilSubscription)
	}
	bus.mu.RLock()
	handlerCount := len(bus.handlers["user.created"])
	bus.mu.RUnlock()
	if handlerCount != 0 {
		t.Fatalf("typed-nil setup left %d handlers", handlerCount)
	}
}

// snapshot reports fixture lifecycle counts without exposing its lock to tests.
func (t *lifecycleTransport) snapshot() (subscribeCalls, subscriptions int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.subscribeCalls, len(t.subscriptions)
}

type lifecycleTransportSubscription struct {
	transport    *lifecycleTransport
	inflight     sync.WaitGroup
	closeOnce    sync.Once
	closeErr     error
	closeStarted chan struct{}
}

// Close waits for callbacks and removes this fixture subscription exactly once.
func (s *lifecycleTransportSubscription) Close() error {
	s.closeOnce.Do(func() {
		close(s.closeStarted)
		s.inflight.Wait()
		s.transport.mu.Lock()
		for i, candidate := range s.transport.subscriptions {
			if candidate == s {
				s.transport.subscriptions = append(s.transport.subscriptions[:i], s.transport.subscriptions[i+1:]...)
				s.transport.handlers = append(s.transport.handlers[:i], s.transport.handlers[i+1:]...)
				break
			}
		}
		s.transport.mu.Unlock()
	})
	return s.closeErr
}

// TestConcurrentTransportSubscribeSharesOneSetup verifies one backend subscription serves concurrent local handlers.
func TestConcurrentTransportSubscribeSharesOneSetup(t *testing.T) {
	transport := &lifecycleTransport{
		subscribeStart: make(chan struct{}),
		subscribeGate:  make(chan struct{}),
	}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	type result struct {
		sub Subscription
		err error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			sub, subscribeErr := bus.Subscribe(func(userCreated) {})
			results <- result{sub: sub, err: subscribeErr}
		}()
	}
	waitForSignal(t, transport.subscribeStart, "first transport setup")
	waitForCondition(t, func() bool {
		bus.mu.RLock()
		defer bus.mu.RUnlock()
		return len(bus.handlers["user.created"]) == 2
	}, "both local handlers to join setup")
	close(transport.subscribeGate)

	var subs []Subscription
	for range 2 {
		result := waitForResult(t, results, "Subscribe result")
		if result.err != nil {
			t.Fatalf("Subscribe returned error: %v", result.err)
		}
		subs = append(subs, result.sub)
	}
	if calls, active := transport.snapshot(); calls != 1 || active != 1 {
		t.Fatalf("transport setup = (%d calls, %d active), want (1, 1)", calls, active)
	}
	if err := subs[0].Close(); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if _, active := transport.snapshot(); active != 1 {
		t.Fatalf("active transport subscriptions after first close = %d, want 1", active)
	}
	if err := subs[1].Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if _, active := transport.snapshot(); active != 0 {
		t.Fatalf("active transport subscriptions after last close = %d, want 0", active)
	}
}

// TestConcurrentTransportSubscribeFailureRollsBackEveryJoiner verifies failure cannot remove the wrong handler.
func TestConcurrentTransportSubscribeFailureRollsBackEveryJoiner(t *testing.T) {
	want := errors.New("setup failed")
	transport := &lifecycleTransport{
		subscribeErr:   want,
		subscribeStart: make(chan struct{}),
		subscribeGate:  make(chan struct{}),
	}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, subscribeErr := bus.Subscribe(func(userCreated) {})
			results <- subscribeErr
		}()
	}
	waitForSignal(t, transport.subscribeStart, "failed transport setup")
	waitForCondition(t, func() bool {
		bus.mu.RLock()
		defer bus.mu.RUnlock()
		return len(bus.handlers["user.created"]) == 2
	}, "both local handlers to join failed setup")
	close(transport.subscribeGate)

	for range 2 {
		if err := waitForResult(t, results, "failed Subscribe result"); !errors.Is(err, want) {
			t.Fatalf("Subscribe error = %v, want %v", err, want)
		}
	}
	bus.mu.RLock()
	handlerCount := len(bus.handlers["user.created"])
	_, hasTopic := bus.transportSubs["user.created"]
	bus.mu.RUnlock()
	if handlerCount != 0 || hasTopic {
		t.Fatalf("failed setup left %d handlers and topic=%v", handlerCount, hasTopic)
	}

	transport.mu.Lock()
	transport.subscribeErr = nil
	transport.subscribeStart = nil
	transport.subscribeGate = nil
	transport.mu.Unlock()
	sub, err := bus.Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("Subscribe after failed setup returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close after retry returned error: %v", err)
	}
}

// TestWaitingTransportSubscribeReturnsExactContextError preserves direct sentinel comparisons on cancellation.
func TestWaitingTransportSubscribeReturnsExactContextError(t *testing.T) {
	transport := &lifecycleTransport{
		subscribeStart: make(chan struct{}),
		subscribeGate:  make(chan struct{}),
	}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	ownerResult := make(chan struct {
		sub Subscription
		err error
	}, 1)
	go func() {
		sub, subscribeErr := bus.Subscribe(func(userCreated) {})
		ownerResult <- struct {
			sub Subscription
			err error
		}{sub: sub, err: subscribeErr}
	}()
	waitForSignal(t, transport.subscribeStart, "owner transport setup")

	ctx, cancel := context.WithCancel(context.Background())
	waiterResult := make(chan error, 1)
	go func() {
		_, subscribeErr := bus.WithContext(ctx).Subscribe(func(userCreated) {})
		waiterResult <- subscribeErr
	}()
	waitForCondition(t, func() bool {
		bus.mu.RLock()
		defer bus.mu.RUnlock()
		return len(bus.handlers["user.created"]) == 2
	}, "cancelable waiter to join setup")
	cancel()
	if err := waitForResult(t, waiterResult, "canceled Subscribe result"); err != context.Canceled {
		t.Fatalf("Subscribe error = %v, want exact %v", err, context.Canceled)
	}

	close(transport.subscribeGate)
	owner := waitForResult(t, ownerResult, "owner Subscribe result")
	if owner.err != nil {
		t.Fatalf("owner Subscribe returned error: %v", owner.err)
	}
	if err := owner.sub.Close(); err != nil {
		t.Fatalf("owner Close returned error: %v", err)
	}
}

// TestImmediateTransportSubscribeFailureAllowsRetry covers failure before another subscriber can join.
func TestImmediateTransportSubscribeFailureAllowsRetry(t *testing.T) {
	want := errors.New("setup failed immediately")
	transport := &lifecycleTransport{subscribeErr: want}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := bus.Subscribe(func(userCreated) {}); !errors.Is(err, want) {
		t.Fatalf("Subscribe error = %v, want %v", err, want)
	}
	transport.mu.Lock()
	transport.subscribeErr = nil
	transport.mu.Unlock()
	sub, err := bus.Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("Subscribe retry returned error: %v", err)
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// TestSubscriptionCloseAllowsCallbackReentrancy proves external close never runs under the bus lock.
func TestSubscriptionCloseAllowsCallbackReentrancy(t *testing.T) {
	transport := &lifecycleTransport{}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	handlerStarted := make(chan struct{})
	allowReentry := make(chan struct{})
	handlerDone := make(chan error, 1)
	sub, err := bus.Subscribe(func(userCreated) {
		close(handlerStarted)
		<-allowReentry
		reentrant, subscribeErr := bus.Subscribe(func(userCreated) {})
		if subscribeErr == nil {
			subscribeErr = reentrant.Close()
		}
		handlerDone <- subscribeErr
	})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	publishDone := make(chan error, 1)
	go func() {
		publishDone <- bus.Publish(userCreated{})
	}()
	waitForSignal(t, handlerStarted, "handler entry")

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- sub.Close()
	}()
	transport.mu.Lock()
	firstSub := transport.subscriptions[0]
	transport.mu.Unlock()
	waitForSignal(t, firstSub.closeStarted, "transport close")
	close(allowReentry)

	if err := waitForResult(t, handlerDone, "reentrant handler"); err != nil {
		t.Fatalf("reentrant handler returned error: %v", err)
	}
	if err := waitForResult(t, publishDone, "Publish"); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if err := waitForResult(t, closeDone, "Close"); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

// TestSubscriptionCloseReturnsStableTransportError verifies teardown failures are observable and idempotent.
func TestSubscriptionCloseReturnsStableTransportError(t *testing.T) {
	want := errors.New("close failed")
	transport := &lifecycleTransport{closeErr: want}
	bus, err := New(Config{Transport: transport})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	sub, err := bus.Subscribe(func(userCreated) {})
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}
	for range 2 {
		if err := sub.Close(); !errors.Is(err, want) {
			t.Fatalf("Close error = %v, want %v", err, want)
		}
	}
}

// waitForSignal bounds lifecycle synchronization so a lock regression fails instead of hanging CI.
func waitForSignal(t *testing.T, signal <-chan struct{}, operation string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(lifecycleTestDeadline):
		t.Fatalf("timed out waiting for %s", operation)
	}
}

// waitForResult bounds asynchronous lifecycle operations so deadlocks produce actionable failures.
func waitForResult[T any](t *testing.T, results <-chan T, operation string) T {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(lifecycleTestDeadline):
		t.Fatalf("timed out waiting for %s", operation)
		var zero T
		return zero
	}
}

// waitForCondition polls only deterministic in-memory state and applies the same hard deadline.
func waitForCondition(t *testing.T, condition func() bool, operation string) {
	t.Helper()
	deadline := time.Now().Add(lifecycleTestDeadline)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", operation)
}
