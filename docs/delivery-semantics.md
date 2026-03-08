# Delivery Semantics

`events` provides one typed API over two classes of delivery behavior:

- local synchronous dispatch
- distributed pub/sub transport

## Root Semantics

The root `sync` bus is the reference behavior.

- handlers run inline in the caller path
- handler order is deterministic
- the first handler error is returned from `Publish`
- subscription teardown is explicit via `Subscription.Close()`

## Distributed Semantics

Distributed drivers keep the same typed API at the root layer, but their
transport guarantees differ from local dispatch.

Current driver intent:

- `nats`: subject-based fan-out transport
- `redis`: pub/sub fan-out transport
- `kafka`: topic-based fan-out compatibility over a narrower Kafka mapping
- `gcppubsub`: topic/subscription fan-out compatibility over the Pub/Sub emulator

Current distributed drivers are validated against:

- topic routing
- topic override routing
- unsubscribe behavior
- multiple-subscription fan-out behavior

## What events Does Not Promise

`events` does not currently promise:

- durable worker backlog processing
- retry/redelivery contracts
- dead-letter behavior
- queue administration
- consumer-group semantics as a normalized cross-driver API

Those concerns belong in `queue`, not the core `events` contract.

## Product Boundary

Use `events` for pub/sub and event fan-out.

Use `queue` for job processing and worker semantics.
