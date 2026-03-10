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
- Kafka consumer-group semantics, partitioning strategy beyond one partition,
  dedicated SQS queue semantics as a normalized API, and richer Google Pub/Sub
  acknowledgement/retry semantics remain explicit future work.
