<p align="center">
  <img src="./docs/images/logo.png?v=1" width="300" alt="events logo">
</p>

<p align="center">
    events is a typed event bus library for local dispatch and distributed pub/sub.
</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/goforj/events"><img src="https://pkg.go.dev/badge/github.com/goforj/events.svg" alt="Go Reference"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
    <a href="https://github.com/goforj/events/actions"><img src="https://github.com/goforj/events/actions/workflows/test.yml/badge.svg" alt="Go Test"></a>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.25+-blue?logo=go" alt="Go version"></a>
    <img src="https://img.shields.io/github/v/tag/goforj/events?label=version&sort=semver" alt="Latest tag">
    <a href="https://goreportcard.com/report/github.com/goforj/events"><img src="https://goreportcard.com/badge/github.com/goforj/events" alt="Go Report Card"></a>
    <a href="https://codecov.io/gh/goforj/events"><img src="https://codecov.io/gh/goforj/events/graph/badge.svg" alt="Codecov"></a>
<!-- test-count:embed:start -->
  <img src="https://img.shields.io/badge/unit_tests-25-brightgreen" alt="Unit tests (executed count)">
  <img src="https://img.shields.io/badge/integration_tests-62-blue" alt="Integration tests (executed count)">
<!-- test-count:embed:end -->
</p>

## What events is

**events** is a typed event bus for Go and handles **event publication and fan-out**. Durable background work such as retries and worker queues belongs in [`queue`](https://github.com/goforj/queue).

It lets applications publish and subscribe to events using normal Go types, with delivery handled either in-process or through distributed backends like NATS, Redis, Kafka, or Google Pub/Sub.

## Installation

```bash
go get github.com/goforj/events
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/goforj/events"
)

type UserCreated struct {
	ID string `json:"id"`
}

func main() {
	bus, _ := events.NewSync()
	_, _ = bus.Subscribe(func(ctx context.Context, event UserCreated) error {
		fmt.Println("received", event.ID, ctx != nil)
		return nil
	})
	_ = bus.Publish(UserCreated{ID: "123"})
}
```

## Topic Override

```go
type UserCreated struct {
	ID string `json:"id"`
}

func (UserCreated) Topic() string { return "users.created" }
```

## Drivers

Optional distributed backends are separate modules. Keep dependencies lean and install only what you use:

```bash
go get github.com/goforj/events/driver/natsevents
go get github.com/goforj/events/driver/redisevents
go get github.com/goforj/events/driver/kafkaevents
go get github.com/goforj/events/driver/gcppubsubevents
```

## Drivers

|                                                                                                Driver / Backend | Mode | Fan-out | Durable | Queue Semantics | Notes |
|----------------------------------------------------------------------------------------------------------------:| :--- | :---: | :---: | :---: | :--- |
|      <img src="https://img.shields.io/badge/sync-546E7A?logo=go&logoColor=white" alt="Sync"> | In-process | ✓ | x | x | Root-backed synchronous dispatch in the caller path. |
|     <img src="https://img.shields.io/badge/null-9e9e9e?logo=probot&logoColor=white" alt="Null"> | Drop-only | x | x | x | Root-backed no-op transport for disabled eventing and tests. |
|        <img src="https://img.shields.io/badge/nats-27AAE1?logo=natsdotio&logoColor=white" alt="NATS"> | Distributed pub/sub | ✓ | x | x | Subject-based transport with live integration coverage. |
|      <img src="https://img.shields.io/badge/redis-%23DC382D?logo=redis&logoColor=white" alt="Redis"> | Distributed pub/sub | ✓ | x | x | Redis pub/sub transport; Streams are intentionally deferred. |
|      <img src="https://img.shields.io/badge/kafka-231F20?logo=apachekafka&logoColor=white" alt="Kafka"> | Distributed topic/log | ✓ | Partial | x | Current driver validates topic-based fan-out compatibility, not full consumer-group semantics. |
| <img src="https://img.shields.io/badge/gcp%20pub%2Fsub-4285F4?logo=googlecloud&logoColor=white" alt="Google Pub/Sub"> | Distributed topic/subscription | ✓ | Partial | x | Emulator-backed Google Pub/Sub integration with per-subscription fan-out mapping. |
|          <img src="https://img.shields.io/badge/sqs-FF9900?logo=buffer&logoColor=white" alt="SQS"> | Queue target | Planned | ✓ | ✓ | Deferred until a separate async capability surface is intentionally introduced. |

## Driver Constructor Quick Examples

Use root constructors for local backends, and driver-module constructors for
distributed backends. Driver backends live in separate modules so applications
only import/link the optional dependencies they actually use.

```go
package main

import (
	"context"

	"github.com/goforj/events"
	"github.com/goforj/events/driver/gcppubsubevents"
	"github.com/goforj/events/driver/kafkaevents"
	"github.com/goforj/events/driver/natsevents"
	"github.com/goforj/events/driver/redisevents"
)

func main() {
	ctx := context.Background()

	events.NewSync()
	events.NewNull()

	natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
	redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
	kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
	gcppubsubevents.New(ctx, gcppubsubevents.Config{
		ProjectID: "events-project",
		URI:       "127.0.0.1:8085",
	})
}
```

## How It Works

```mermaid
flowchart LR
    A[App publishes typed event] --> B[events.Bus resolves topic and encodes payload]
    B --> C[Driver transports topic plus payload]
    C --> D[Subscriber handler receives decoded typed event]
```

## Benchmarks

Benchmark smoke is intentionally narrow. It tracks the hot in-process paths and, when enabled, a minimal distributed round-trip benchmark through the integration harness.

Normal docs iteration should render from the saved benchmark snapshot, not re-run live backend benchmarks. Use:

```bash
sh scripts/update-docs.sh
```

To refresh the live benchmark snapshot and regenerate the charts:

```bash
sh scripts/refresh-bench-snapshot.sh
```

<!-- bench:embed:start -->
These charts compare one publish-plus-delivery round trip for `sync` and each enabled distributed driver fixture.

Note: `gcppubsub` is excluded from the default charts because the Pub/Sub emulator is not representative enough for backend latency comparison. Benchmark it explicitly with `INTEGRATION_DRIVER=gcppubsub` when needed.

![Events backend latency chart](docs/bench/benchmarks_ns.svg)

![Events backend throughput chart](docs/bench/benchmarks_ops.svg)

![Events backend bytes chart](docs/bench/benchmarks_bytes.svg)

![Events backend allocations chart](docs/bench/benchmarks_allocs.svg)
<!-- bench:embed:end -->

These checks are for obvious regression detection, not for noisy micro-optimism
or hard CI performance gates.

<!-- api:embed:start -->

## API Index

| Group | Functions |
|------:|-----------|
| **Construction** | [events.Codec](#events-codec) [events.Config](#events-config) [events.New](#events-new) [events.NewNull](#events-newnull) [events.NewSync](#events-newsync) [events.Option](#events-option) [events.WithCodec](#events-withcodec) |
| **Core** | [Bus.Driver](#bus-driver) [Bus.Publish](#bus-publish) [Bus.PublishContext](#bus-publishcontext) [Bus.Ready](#bus-ready) [Bus.ReadyContext](#bus-readycontext) [Bus.Subscribe](#bus-subscribe) [Bus.SubscribeContext](#bus-subscribecontext) [events.Bus](#events-bus) [events.Subscription](#events-subscription) [events.TopicEvent](#events-topicevent) |
| **Driver Config** | [gcppubsubevents.Config](#gcppubsubevents-config) [kafkaevents.Config](#kafkaevents-config) [natsevents.Config](#natsevents-config) [redisevents.Config](#redisevents-config) |
| **Driver Constructors** | [gcppubsubevents.New](#gcppubsubevents-new) [kafkaevents.New](#kafkaevents-new) [natsevents.New](#natsevents-new) [redisevents.New](#redisevents-new) |
| **Drivers** | [Driver.Close](#driver-close) [Driver.Driver](#driver-driver) [Driver.PublishContext](#driver-publishcontext) [Driver.Ready](#driver-ready) [Driver.SubscribeContext](#driver-subscribecontext) [gcppubsubevents.Driver](#gcppubsubevents-driver) [kafkaevents.Driver](#kafkaevents-driver) [natsevents.Driver](#natsevents-driver) [redisevents.Driver](#redisevents-driver) |
| **Testing** | [Fake.Bus](#fake-bus) [Fake.Count](#fake-count) [Fake.Records](#fake-records) [Fake.Reset](#fake-reset) [events.Fake](#events-fake) [events.NewFake](#events-newfake) [events.Record](#events-record) |


## Construction

### <a id="events-codec"></a>events.Codec

Codec marshals and unmarshals event payloads.

```go
var codec events.Codec
fmt.Println(codec == nil)
// Output: true
```

### <a id="events-config"></a>events.Config

Config configures root bus construction.

_Example: define bus construction config_

```go
cfg := events.Config{Driver: eventscore.DriverSync}
fmt.Println(cfg.Driver)
// Output: sync
```

_Example: define bus construction config with all fields_

```go
cfg := events.Config{
	Driver:    eventscore.DriverSync, // default: "sync" when empty and no Transport is provided
	Codec:     nil,                   // default: nil uses the built-in JSON codec
	Transport: nil,                   // default: nil keeps dispatch in-process
}
fmt.Println(cfg.Driver)
// Output: sync
```

### <a id="events-new"></a>events.New

New constructs a root bus for the requested driver.

```go
bus, _ := events.New(events.Config{Driver: "sync"})
fmt.Println(bus.Driver())
// Output: sync
```

### <a id="events-newnull"></a>events.NewNull

NewNull constructs the root null bus.

```go
bus, _ := events.NewNull()
fmt.Println(bus.Driver())
// Output: null
```

### <a id="events-newsync"></a>events.NewSync

NewSync constructs the root sync bus.

```go
bus, _ := events.NewSync()
fmt.Println(bus.Driver())
// Output: sync
```

### <a id="events-option"></a>events.Option

Option configures root bus behavior.

```go
opt := events.WithCodec(nil)
fmt.Println(opt != nil)
// Output: true
```

### <a id="events-withcodec"></a>events.WithCodec

WithCodec overrides the default event codec.

```go
bus, _ := events.NewSync(events.WithCodec(nil))
fmt.Println(bus.Driver())
// Output: sync
```

## Core

### <a id="bus-driver"></a>Bus.Driver

Driver reports the active backend.

```go
bus, _ := events.NewSync()
fmt.Println(bus.Driver())
// Output: sync
```

### <a id="bus-publish"></a>Bus.Publish

Publish publishes an event using the background context.

```go
type UserCreated struct {
	ID string `json:"id"`
}

bus, _ := events.NewSync()
_, _ = bus.Subscribe(func(event UserCreated) {
	fmt.Println(event.ID)
})
_ = bus.Publish(UserCreated{ID: "123"})
// Output: 123
```

### <a id="bus-publishcontext"></a>Bus.PublishContext

PublishContext publishes an event using the configured codec and dispatch flow.

```go
type UserCreated struct {
	ID string `json:"id"`
}

bus, _ := events.NewSync()
_, _ = bus.Subscribe(func(ctx context.Context, event UserCreated) error {
	fmt.Println(event.ID, ctx != nil)
	return nil
})
_ = bus.PublishContext(context.Background(), UserCreated{ID: "123"})
// Output: 123 true
```

### <a id="bus-ready"></a>Bus.Ready

Ready reports whether the bus is ready.

```go
bus, _ := events.NewSync()
fmt.Println(bus.Ready() == nil)
// Output: true
```

### <a id="bus-readycontext"></a>Bus.ReadyContext

ReadyContext reports whether the bus is ready.

```go
bus, _ := events.NewSync()
fmt.Println(bus.ReadyContext(context.Background()) == nil)
// Output: true
```

### <a id="bus-subscribe"></a>Bus.Subscribe

Subscribe registers a handler using the background context.

```go
type UserCreated struct {
	ID string `json:"id"`
}

bus, _ := events.NewSync()
sub, _ := bus.Subscribe(func(ctx context.Context, event UserCreated) error {
	_ = ctx
	_ = event
	return nil
})
defer sub.Close()
```

### <a id="bus-subscribecontext"></a>Bus.SubscribeContext

SubscribeContext registers a typed handler.

```go
type UserCreated struct {
	ID string `json:"id"`
}

bus, _ := events.NewSync()
sub, _ := bus.SubscribeContext(context.Background(), func(ctx context.Context, event UserCreated) error {
	_ = ctx
	_ = event
	return nil
})
defer sub.Close()
```

### <a id="events-bus"></a>events.Bus

Bus is the root event bus implementation.

```go
bus, _ := events.NewSync()
var root *events.Bus = bus
fmt.Println(root.Driver())
// Output: sync
```

### <a id="events-subscription"></a>events.Subscription

Subscription releases a subscription when closed.

```go
type UserCreated struct {
	ID string `json:"id"`
}

bus, _ := events.NewSync()
sub, _ := bus.Subscribe(func(UserCreated) {})
fmt.Println(sub.Close() == nil)
// Output: true
```

### <a id="events-topicevent"></a>events.TopicEvent

TopicEvent overrides the derived topic for an event.

```go
var event events.TopicEvent
fmt.Println(event == nil)
// Output: true
```

## Driver Config

### <a id="gcppubsubevents-config"></a>gcppubsubevents.Config

Config configures Google Pub/Sub transport construction.

_Example: define Google Pub/Sub driver config_

```go
cfg := gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
}
fmt.Println(cfg.ProjectID)
// Output: events-project
```

_Example: define Google Pub/Sub driver config with all fields_

```go
cfg := gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085", // default: "" is invalid unless Client is provided
	Client:    nil,              // default: nil creates a client from ProjectID and URI
}
fmt.Println(cfg.ProjectID)
// Output: events-project
```

### <a id="kafkaevents-config"></a>kafkaevents.Config

Config configures Kafka transport construction.

_Example: define Kafka driver config_

```go
cfg := kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}}
fmt.Println(cfg.Brokers[0])
// Output: 127.0.0.1:9092
```

_Example: define Kafka driver config with all fields_

```go
cfg := kafkaevents.Config{
	Brokers: []string{"127.0.0.1:9092"},
	Dialer:  nil, // default: nil uses a zero-value kafka.Dialer
	Writer:  nil, // default: nil builds a writer with single-message, auto-topic defaults
}
fmt.Println(cfg.Brokers[0])
// Output: 127.0.0.1:9092
```

### <a id="natsevents-config"></a>natsevents.Config

Config configures NATS transport construction.

_Example: define NATS driver config_

```go
cfg := natsevents.Config{URL: "nats://127.0.0.1:4222"}
fmt.Println(cfg.URL)
// Output: nats://127.0.0.1:4222
```

_Example: define NATS driver config with all fields_

```go
cfg := natsevents.Config{
	URL:  "nats://127.0.0.1:4222",
	Conn: nil, // default: nil dials URL instead of reusing an existing connection
}
fmt.Println(cfg.URL)
// Output: nats://127.0.0.1:4222
```

### <a id="redisevents-config"></a>redisevents.Config

Config configures Redis transport construction.

_Example: define Redis driver config_

```go
cfg := redisevents.Config{Addr: "127.0.0.1:6379"}
fmt.Println(cfg.Addr)
// Output: 127.0.0.1:6379
```

_Example: define Redis driver config with all fields_

```go
cfg := redisevents.Config{
	Addr:   "127.0.0.1:6379",
	Client: nil, // default: nil constructs a client from Addr
}
fmt.Println(cfg.Addr)
// Output: 127.0.0.1:6379
```

## Driver Constructors

### <a id="gcppubsubevents-new"></a>gcppubsubevents.New

New constructs a Google Pub/Sub-backed driver.

```go
driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
})
fmt.Println(driver != nil)
// Output: true
```

### <a id="kafkaevents-new"></a>kafkaevents.New

New constructs a Kafka-backed driver.

```go
driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
fmt.Println(driver != nil)
// Output: true
```

### <a id="natsevents-new"></a>natsevents.New

New connects a NATS-backed driver from config.

```go
driver, _ := natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
fmt.Println(driver != nil)
// Output: true
```

### <a id="redisevents-new"></a>redisevents.New

New constructs a Redis pub/sub-backed driver.

```go
driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
fmt.Println(driver != nil)
// Output: true
```

## Drivers

### <a id="driver-close"></a>Driver.Close

Close closes the underlying Pub/Sub client.

_Example: close a NATS driver_

```go
driver, _ := natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
fmt.Println(driver.Close() == nil)
// Output: true
```

_Example: close a Redis driver_

```go
driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
fmt.Println(driver.Close() == nil)
// Output: true
```

_Example: close a Kafka driver_

```go
driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
fmt.Println(driver.Close() == nil)
// Output: true
```

_Example: close a Google Pub/Sub driver_

```go
driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
})
fmt.Println(driver.Close() == nil)
// Output: true
```

### <a id="driver-driver"></a>Driver.Driver

Driver reports the active backend kind.

_Example: inspect the driver kind_

```go
driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
fmt.Println(driver.Driver())
// Output: redis
```

_Example: inspect the driver kind_

```go
driver, _ := natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
fmt.Println(driver.Driver())
// Output: nats
```

_Example: inspect the driver kind_

```go
driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
fmt.Println(driver.Driver())
// Output: kafka
```

_Example: inspect the driver kind_

```go
driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
})
fmt.Println(driver.Driver())
// Output: gcppubsub
```

### <a id="driver-publishcontext"></a>Driver.PublishContext

PublishContext publishes a topic payload to Google Pub/Sub.

_Example: publish a raw message through Redis_

```go
driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
_ = driver.PublishContext(context.Background(), eventscore.Message{
	Topic:   "users.created",
	Payload: []byte(`{"id":"123"}`),
})
```

_Example: publish a raw message through NATS_

```go
driver, _ := natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
_ = driver.PublishContext(context.Background(), eventscore.Message{
	Topic:   "users.created",
	Payload: []byte(`{"id":"123"}`),
})
```

_Example: publish a raw message through Kafka_

```go
driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
_ = driver.PublishContext(context.Background(), eventscore.Message{
	Topic:   "users.created",
	Payload: []byte(`{"id":"123"}`),
})
```

_Example: publish a raw message through Google Pub/Sub_

```go
driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
})
_ = driver.PublishContext(context.Background(), eventscore.Message{
	Topic:   "users.created",
	Payload: []byte(`{"id":"123"}`),
})
```

### <a id="driver-ready"></a>Driver.Ready

Ready checks Google Pub/Sub connectivity.

_Example: check Redis connectivity_

```go
driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
fmt.Println(driver.Ready(context.Background()) == nil)
// Output: true
```

_Example: check NATS connectivity_

```go
driver, _ := natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
fmt.Println(driver.Ready(context.Background()) == nil)
// Output: true
```

_Example: check Kafka connectivity_

```go
driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
fmt.Println(driver.Ready(context.Background()) == nil)
// Output: true
```

_Example: check Google Pub/Sub connectivity_

```go
driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
})
fmt.Println(driver.Ready(context.Background()) == nil)
// Output: true
```

### <a id="driver-subscribecontext"></a>Driver.SubscribeContext

SubscribeContext subscribes to a Google Pub/Sub topic and forwards messages.

_Example: subscribe to a Redis channel_

```go
driver, _ := redisevents.New(redisevents.Config{Addr: "127.0.0.1:6379"})
sub, _ := driver.SubscribeContext(context.Background(), "users.created", func(ctx context.Context, msg eventscore.Message) error {
	_ = ctx
	_ = msg
	return nil
})
fmt.Println(sub != nil)
// Output: true
```

_Example: subscribe to a raw NATS subject_

```go
driver, _ := natsevents.New(natsevents.Config{URL: "nats://127.0.0.1:4222"})
sub, _ := driver.SubscribeContext(context.Background(), "users.created", func(ctx context.Context, msg eventscore.Message) error {
	_ = ctx
	_ = msg
	return nil
})
fmt.Println(sub != nil)
// Output: true
```

_Example: subscribe to a Kafka topic_

```go
driver, _ := kafkaevents.New(kafkaevents.Config{Brokers: []string{"127.0.0.1:9092"}})
sub, _ := driver.SubscribeContext(context.Background(), "users.created", func(ctx context.Context, msg eventscore.Message) error {
	_ = ctx
	_ = msg
	return nil
})
fmt.Println(sub != nil)
// Output: true
```

_Example: subscribe to a Google Pub/Sub topic_

```go
driver, _ := gcppubsubevents.New(context.Background(), gcppubsubevents.Config{
	ProjectID: "events-project",
	URI:       "127.0.0.1:8085",
})
sub, _ := driver.SubscribeContext(context.Background(), "users.created", func(ctx context.Context, msg eventscore.Message) error {
	_ = ctx
	_ = msg
	return nil
})
fmt.Println(sub != nil)
// Output: true
```

### <a id="gcppubsubevents-driver"></a>gcppubsubevents.Driver

Driver is a Google Pub/Sub-backed events transport.

```go
var driver *gcppubsubevents.Driver
fmt.Println(driver == nil)
// Output: true
```

### <a id="kafkaevents-driver"></a>kafkaevents.Driver

Driver is a Kafka-backed events transport.

```go
var driver *kafkaevents.Driver
fmt.Println(driver == nil)
// Output: true
```

### <a id="natsevents-driver"></a>natsevents.Driver

Driver is a NATS-backed events transport.

```go
var driver *natsevents.Driver
fmt.Println(driver == nil)
// Output: true
```

### <a id="redisevents-driver"></a>redisevents.Driver

Driver is a Redis pub/sub-backed events transport.

```go
var driver *redisevents.Driver
fmt.Println(driver == nil)
// Output: true
```

## Testing

### <a id="fake-bus"></a>Fake.Bus

Bus returns the wrapped API to inject into code under test.

```go
fake := events.NewFake()
bus := fake.Bus()
fmt.Println(bus.Ready() == nil)
// Output: true
```

### <a id="fake-count"></a>Fake.Count

Count returns the total number of recorded publishes.

```go
type UserCreated struct {
	ID string `json:"id"`
}

fake := events.NewFake()
_ = fake.Bus().Publish(UserCreated{ID: "123"})
fmt.Println(fake.Count())
// Output: 1
```

### <a id="fake-records"></a>Fake.Records

Records returns a copy of recorded publishes.

```go
type UserCreated struct {
	ID string `json:"id"`
}

fake := events.NewFake()
_ = fake.Bus().Publish(UserCreated{ID: "123"})
fmt.Println(len(fake.Records()))
// Output: 1
```

### <a id="fake-reset"></a>Fake.Reset

Reset clears recorded publishes.

```go
type UserCreated struct {
	ID string `json:"id"`
}

fake := events.NewFake()
_ = fake.Bus().Publish(UserCreated{ID: "123"})
fake.Reset()
fmt.Println(fake.Count())
// Output: 0
```

### <a id="events-fake"></a>events.Fake

Fake provides a root-package testing helper that records published events.

```go
fake := events.NewFake()
fmt.Println(fake.Count())
// Output: 0
```

### <a id="events-newfake"></a>events.NewFake

NewFake creates a new fake event harness backed by the root sync bus.

```go
fake := events.NewFake()
fmt.Println(fake.Count())
// Output: 0
```

### <a id="events-record"></a>events.Record

Record captures one published event observed by a Fake bus.

```go
type UserCreated struct {
	ID string `json:"id"`
}

record := events.Record{Event: UserCreated{ID: "123"}}
fmt.Printf("%T\n", record.Event)
// Output: main.UserCreated
```
<!-- api:embed:end -->

## Docs Tooling

The repository includes lightweight docs tooling under `docs/`.

Fast docs loop:

```bash
sh scripts/update-docs.sh
```

Refresh the live benchmark snapshot and regenerate the charts:

```bash
sh scripts/refresh-bench-snapshot.sh
```

Update generated README sections and validate required structure:

```bash
go run ./docs/readme/main.go
```

Update executed test-count badges:

```bash
go run ./docs/readme/testcounts/main.go
```

Generate example programs used by docs checks:

```bash
go run ./docs/examplegen/main.go
```

Rerun the fast docs update loop locally:

```bash
sh docs/watcher.sh
```

## Release Tagging

This repository uses per-module tags. Use `scripts/tag-all-modules.sh vX.Y.Z` to apply the root tag plus all driver-module tags in one command. 
