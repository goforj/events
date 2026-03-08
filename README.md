# events

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
    <img src="https://img.shields.io/badge/unit_tests-93-brightgreen" alt="Unit tests (executed count)">
    <img src="https://img.shields.io/badge/integration_tests-62-blue" alt="Integration tests (executed count)">
<!-- test-count:embed:end -->
</p>

## What events is

`events` is a typed event bus for Go.

It lets applications publish and subscribe to events using normal Go types, with delivery handled either in-process or through distributed backends like NATS, Redis, Kafka, or Google Pub/Sub.

`events` handles **event publication and fan-out**. Durable background work such as retries and worker queues belongs in [`queue`](https://github.com/goforj/queue).

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

Benchmark smoke is intentionally narrow. It tracks the hot in-process paths and,
when enabled, a minimal distributed round-trip benchmark through the
integration harness.

Normal docs iteration should render from the saved benchmark snapshot, not
re-run live backend benchmarks. Use:

```bash
sh scripts/update-docs.sh
```

To refresh the live benchmark snapshot and regenerate the charts:

```bash
sh scripts/refresh-bench-snapshot.sh
```

<!-- bench:embed:start -->
_generated by `go test -tags=benchrender ./docs/bench -run TestRenderBenchmarks`_

Core hot-path benchmarks track the in-process event bus overhead directly.
Backend round-trip benchmarks compare the local sync bus against every enabled broker-backed driver fixture.

| Benchmark | ns/op | ops/s | B/op | allocs/op |
|:----------|-----:|-----:|-----:|---------:|
| `ResolveTopic` | 157.3 | 6357279 | 96 | 6 |
| `NewRegisteredHandler` | 182.9 | 5467469 | 96 | 6 |
| `PublishNoSubscribers` | 198.9 | 5027652 | 104 | 7 |
| `PublishOneSubscriber` | 364.4 | 2744237 | 336 | 10 |
| `SyncPublishRoundTrip` | 380.1 | 2630887 | 336 | 10 |
| `PublishMultipleSubscribers` | 831.3 | 1202935 | 1000 | 16 |

### Backend Round-Trip by Driver

These charts compare one publish-plus-delivery round trip for `sync` and each enabled distributed driver fixture.

Note: `gcppubsub` is excluded from the default charts because the Pub/Sub emulator is not representative enough for backend latency comparison. Benchmark it explicitly with `INTEGRATION_DRIVER=gcppubsub` when needed.

![Events backend latency chart](docs/bench/benchmarks_ns.svg)

![Events backend throughput chart](docs/bench/benchmarks_ops.svg)

![Events backend bytes chart](docs/bench/benchmarks_bytes.svg)

![Events backend allocations chart](docs/bench/benchmarks_allocs.svg)
<!-- bench:embed:end -->

These checks are for obvious regression detection, not for noisy micro-optimism
or hard CI performance gates.

## Docs Tooling

The repository includes lightweight docs tooling under `docs/`:

- `sh scripts/update-docs.sh` is the fast docs loop; it renders benchmark charts from the saved snapshot
- `sh scripts/refresh-bench-snapshot.sh` refreshes the live benchmark snapshot and regenerates the charts
- `go run ./docs/readme` updates generated README sections and validates required structure
- `go run ./docs/readme/testcounts` updates executed test-count badges
- `go run ./docs/examplegen` lists example programs used by docs checks
- `sh docs/watcher.sh` reruns the fast docs update loop locally

## API Index

<!-- api:embed:start -->
_generated by `go run ./docs/readme`_

### Core

| Package | API | Purpose |
| --- | --- | --- |
| `events` | `New`, `NewSync`, `NewNull`, `WithCodec` | Root constructors and options for the typed bus. |
| `events` | `Bus.Publish`, `Bus.PublishContext` | Publish typed events through the configured transport. |
| `events` | `Bus.Subscribe`, `Bus.SubscribeContext` | Register typed handlers and receive a closable subscription. |
| `events` | `Bus.Ready`, `Bus.ReadyContext`, `Bus.Driver`, `Bus.Close` | Transport health, backend identity, and shutdown. |
| `events` | `NewFake` | Lightweight root fake for application tests. |
| `eventscore` | `Driver`, `Message`, `DriverAPI`, `Subscription` | Driver-facing transport contracts. |
| `eventstest` | `RunBusContract`, `RunNullBusContract` | Shared contract harness for bus implementations. |
| `eventsfake` | `New` | Assertion-oriented fake/testing helper module. |

### Driver Modules

| Package | Constructor | Purpose |
| --- | --- | --- |
| `driver/natsevents` | `natsevents.New` | NATS subject-based distributed fan-out transport. |
| `driver/redisevents` | `redisevents.New` | Redis pub/sub distributed fan-out transport. |
| `driver/kafkaevents` | `kafkaevents.New` | Kafka topic-based fan-out compatibility transport. |
| `driver/gcppubsubevents` | `gcppubsubevents.New` | Google Pub/Sub emulator-backed transport. |
<!-- api:embed:end -->

## Release Tagging

This repository uses per-module tags. Use `scripts/tag-all-modules.sh vX.Y.Z` to
apply the root tag plus matching module tags for:

- `eventscore`
- `eventstest`
- `eventsfake`
- `examples`
- `docs`
- `driver/gcppubsubevents`
- `driver/kafkaevents`
- `driver/natsevents`
- `driver/redisevents`
- `integration`
