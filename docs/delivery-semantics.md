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
- context-bound handles share the same bus state; deriving a handle does not
  clone subscriptions or synchronization
- concurrent subscriptions for one topic share one transport setup, and all
  callers observe the same setup failure
- the last local subscription closes its transport subscription without
  holding the bus registry lock; a close error is returned and remains stable
  across repeated `Close` calls

## Distributed Semantics

Distributed drivers keep the same typed API at the root layer, but their
transport guarantees differ from local dispatch.

Current driver intent:

- `nats`: subject-based fan-out transport
- `natsjetstream`: stream-backed fan-out compatibility via ephemeral JetStream consumers
- `redis`: pub/sub fan-out transport
- `kafka`: topic-based fan-out compatibility over a narrower Kafka mapping
- `sns`: topic fan-out compatibility via dedicated SQS subscriptions
- `gcppubsub`: topic/subscription fan-out compatibility over the Pub/Sub emulator

Current distributed drivers are validated against:

- topic routing
- topic override routing
- unsubscribe behavior
- multiple-subscription fan-out behavior

Driver callbacks forward handler errors to the transport callback boundary,
but the current distributed drivers intentionally do not normalize retries or
redelivery from those errors. Backend acknowledgement behavior is therefore a
driver property, not a portable `events` guarantee.

`Subscribe` suppresses callbacks until the transport setup handshake has
completed and ignores callbacks from a retired subscription generation. This
prevents a closing backend callback from reaching handlers registered against
a replacement subscription.

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
