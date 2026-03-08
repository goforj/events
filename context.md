# Events Context Handoff

## Repo

- Path: `/Users/cmiles/code/events`
- Product: GoForj `events`
- Scope: typed event bus with local sync dispatch and distributed pub/sub

## Product Boundary

`events` is for:

- typed event publication
- typed subscriptions
- local synchronous dispatch
- distributed pub/sub fan-out

`events` is not for:

- worker/runtime queue semantics
- retries/redelivery guarantees
- dead-letter behavior
- backlog draining
- queue administration

Those concerns belong in `queue`.

## Core Decisions

- Public API is type-based.
- Driver API is topic-based.
- Root `sync` bus is the reference contract.
- `Publish` is synchronous by contract for local sync dispatch.
- Distributed drivers provide async transport/pub/sub behavior, but not queue semantics.
- Redis support is intentionally `pub/sub` first.
- Redis Streams are deferred until an explicit async capability surface exists.
- SQS is deferred until an explicit async capability surface exists.

## Implemented Backends

Root-backed:

- `sync`
- `null`

Distributed drivers:

- `driver/natsevents`
- `driver/redisevents`
- `driver/kafkaevents`
- `driver/gcppubsubevents`

## Important Driver Notes

### Redis

- current driver is Redis pub/sub
- not Streams
- intended as distributed fan-out transport

### Kafka

- current driver is intentionally narrow
- creates single-partition topics
- subscribes by reading partition `0` directly
- validates fan-out/topic compatibility, not full consumer-group semantics

### Google Pub/Sub

- emulator-backed integration
- current mapping creates a unique Pub/Sub subscription per bus subscription
- this preserves current fan-out behavior under the events contract

## Modules

- `events`
- `eventscore`
- `eventstest`
- `eventsfake`
- `integration`
- `examples`
- `docs`

## Integration State

Centralized live integration exists under `integration/`.

Shared distributed scenario harness:

- `integration/scenario/driver_contract.go`

Current shared distributed scenarios:

- `ready`
- `publish_subscribe`
- `unsubscribe`
- `fanout_subscriptions`
- `topic_override`

Live integration packages:

- `integration/root`
- `integration/nats`
- `integration/redis`
- `integration/kafka`
- `integration/gcppubsub`
- `integration/all`
- `integration/testenv`
- `integration/bench`

Testcontainer-backed env helpers exist for:

- NATS
- Redis
- Kafka
- Google Pub/Sub emulator

## Coverage / Validation

Important scripts:

- `sh scripts/test-all-modules.sh`
- `sh scripts/check-docs.sh`
- `sh scripts/update-docs.sh`
- `sh scripts/check-bench-smoke.sh`
- `sh scripts/test-integration.sh`
- `RUN_INTEGRATION=1 sh scripts/coverage-codecov.sh`

Important coverage behavior:

- `scripts/coverage-codecov.sh` uses integration `-coverpkg` when `RUN_INTEGRATION=1`
- this lets live distributed integration contribute to broker package coverage

Recent headline numbers:

- merged live coverage reached `84.6%`
- root package coverage reached `93.0%`
- Kafka direct package coverage improved to `52.4%`

## Docs / README Tooling

README is hand-authored with generated marker-owned sections.

Current markers in `README.md`:

- `<!-- test-count:embed:start -->`
- `<!-- bench:embed:start -->`
- `<!-- api:embed:start -->`

Docs tooling:

- `docs/readme/main.go`
  - updates the API section
  - validates required README structure
- `docs/readme/testcounts/main.go`
  - updates executed test-count badges
- `docs/examplegen/main.go`
  - lists example directories for docs checks
- `docs/bench/render.go`
  - updates the benchmark marker section from a snapshot or fresh benchmark run

Docs scripts:

- `scripts/update-docs.sh`
  - test-count badges
  - README API section
  - example listing
  - benchmark section in render-only mode
- `scripts/check-docs.sh`
  - verifies the same docs surfaces
- `docs/watcher.sh`
  - delegates to `scripts/update-docs.sh`

Additional docs:

- `docs/compatibility.md`
- `docs/delivery-semantics.md`

## Benchmark Docs

`docs/bench` now exists with:

- `render.go`
- `render_test.go`
- `benchmarks_rows.json`

Current snapshot includes:

- `ResolveTopic`
- `NewRegisteredHandler`
- `PublishNoSubscribers`
- `PublishOneSubscriber`
- `PublishMultipleSubscribers`
- `DistributedPublishRoundTrip/nats`

This is intentionally lighter than the full SVG/chart pipeline used in `cache`
and `queue`, but it establishes the same structural pattern:

- saved benchmark snapshot
- marker-owned README section
- docs verification hook

## Known Rough Edge

`docs/readme/testcounts` tries to collect executed integration test counts.

When Docker/testcontainers are unavailable, it:

- keeps the existing integration badge value
- emits a short warning

This is acceptable for now, but if more polish is needed later, likely options are:

- make integration count refresh opt-in
- or count only non-Docker integration subsets by default

## README State

README now includes:

- product intro + badges
- installation
- driver matrix
- driver constructor quick examples
- module layout table
- quick start
- topic override
- delivery semantics
- `events vs queue`
- validation
- benchmarks with generated marker block
- docs tooling
- API index with generated marker block
- release tagging

## Likely Next Step

If continuing from here, the best next move is a release/stabilization pass:

1. Run:
   - `sh scripts/test-all-modules.sh`
   - `sh scripts/check-docs.sh`
   - `sh scripts/check-bench-smoke.sh`
   - `sh scripts/test-integration.sh`
   - `RUN_INTEGRATION=1 sh scripts/coverage-codecov.sh`
2. Run:
   - `sh scripts/update-docs.sh`
3. Review any final README/docs wording
4. Tag with:
   - `scripts/tag-all-modules.sh vX.Y.Z`

## Deferred Scope

- observer/metrics surface
- Redis Streams
- SQS
- any richer async/worker contract

Do not blur `events` into `queue`.
