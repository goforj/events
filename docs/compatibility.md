# Compatibility

`events` aims to keep the public API type-based and predictable across backends,
but transport behavior is not perfectly identical across all implementations.

## Current Backends

| Backend | Delivery Style | Local/Distributed | Notes |
| --- | --- | --- | --- |
| `sync` | synchronous | local | Handlers run inline in the caller path. |
| `null` | drop-only | local | Publish succeeds without delivery. |
| `gcppubsub` | topic/subscription | distributed | Emulator-backed topic plus per-subscription fan-out mapping. |
| `kafka` | topic/log | distributed | Single-partition topic creation with direct partition reads. |
| `natsjetstream` | stream/consumer | distributed | JetStream-backed stream retention with one ephemeral consumer per subscription. |
| `nats` | pub/sub | distributed | Subject-based fan-out transport. |
| `redis` | pub/sub | distributed | Channel-based fan-out transport. |
| `sns` | topic plus queue | distributed | SNS fan-out with one SQS queue per bus subscription. |

## Semantic Notes

- `sync` is the reference behavior for in-process dispatch.
- `null` preserves API shape but intentionally does not deliver events.
- `gcppubsub` currently maps each events topic to a Pub/Sub topic and creates a
  unique Pub/Sub subscription per bus subscription so the current contract
  behaves like fan-out delivery.
- `kafka` currently maps each events topic to a single-partition Kafka topic
  and reads partition `0` directly for each subscription so the current
  contract behaves like fan-out delivery.
- `natsjetstream` currently maps each events topic to a JetStream stream and
  creates an ephemeral pull consumer per bus subscription so subscribe/close
  remains explicit while stream durability stays opt-in at the driver layer.
- Kafka support is intentionally narrower than native Kafka capabilities:
  the current driver validates topic-based fan-out compatibility, not consumer
  groups, replay controls, or multi-partition routing semantics.
- `nats` and `redis` currently model distributed fan-out transport, not durable
  queue semantics.
- `sns` currently maps each events topic to an SNS topic and creates a
  dedicated SQS queue per bus subscription so the current contract behaves
  like fan-out delivery instead of shared-consumer queue semantics.
- Redis uses pub/sub in the current driver. Redis Streams are intentionally
  deferred until async delivery semantics become a first-class API surface.
- Event payload bytes remain codec-owned, and the default codec uses JSON.
  Distributed transports use native metadata where available; Redis framing
  requires an explicit opt-in as described below.
- Local construction accepts only `sync` and `null` without a transport.
  Distributed driver names require a matching `Config.Transport`, preventing
  a configured distributed backend from silently behaving like the sync bus.
- Nil contexts normalize to `context.Background()`. Typed-nil codecs and
  transports are rejected during construction instead of failing on first use.
- Distributed handler errors do not currently define a portable ack, retry, or
  redelivery contract. Each backend keeps the behavior documented by its
  driver until such semantics become a first-class API surface.
- Kafka consumer-group semantics, partitioning strategy beyond one partition,
  dedicated SQS queue semantics as a normalized API, and richer Google Pub/Sub
  acknowledgement/retry semantics remain explicit future work.


## Optional message context

Message headers and `WithContextPropagator` are optional. Without a propagator, publication remains unchanged. Native-header transports retain their exact application body, and old consumers can ignore the additional native metadata.

Propagator carriers are limited to 10 fields and 8 KiB of both raw key/value bytes and JSON-encoded bytes on publication and receipt. Oversized carriers and invalid UTF-8 are ignored while the event payload is delivered normally. The field limit accommodates SNS raw SQS delivery. Direct SNS driver publications with more than 10 `Message.Headers` entries return an error before contacting SNS.

Google Pub/Sub applies [native attribute limits](https://docs.cloud.google.com/pubsub/docs/publisher#use_attributes_to_publish_a_message): at most 100 attributes, keys up to 256 bytes, values up to 1024 bytes, and no keys starting with lowercase `goog`. Its adapter omits the entire optional carrier if any field violates these rules, preserving the payload bytes. The shared carrier budget remains unchanged for other transports.

Redis pub/sub uses an explicitly enabled metadata envelope because the protocol has no message headers. Deploy consumers configured with `redisevents.Config.EnableHeaders=true` before enabling propagating publishers. Opted-in consumers accept ordinary legacy JSON publications and new framed messages. External Redis subscribers need the same envelope decoding when consuming header-bearing publications.

### Redis header framing

`redisevents.Config.EnableHeaders` defaults to false. This preserves arbitrary raw payloads, including bytes beginning with the metadata prefix. Publishing headers without enabling framing returns an error before Redis I/O.

Enable framing on every subscriber before publishing headers on those channels. Enabled transports preserve ordinary JSON publications and escape prefix-bearing binary payloads from updated publishers. In this mode, the byte prefix `\x00goforj-event\x01` is reserved. Legacy binary publishers whose payloads begin with that prefix must be upgraded or moved to separate channels before consumers opt in.

### Struct literals

Message metadata adds `Headers` to `eventscore.Message`, and Redis configuration adds `EnableHeaders` to `redisevents.Config`. Keyed literals continue to compile. Positional literals must name their fields when upgrading, for example `eventscore.Message{Topic: topic, Payload: payload}` and `redisevents.Config{Addr: addr, Client: client}`. Headers belong to the shared transport message so every driver can preserve them independently from the payload; the Redis flag keeps framing an explicit channel-level choice.
