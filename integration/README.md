# integration

Centralized integration tests for `events` backends live here.

Current scope:

- root `sync` baseline integration coverage
- root `null` baseline integration coverage

Future scope:

- containerized broker-backed integration coverage for NATS, Redis, Kafka, and
  emulator-backed cloud transports

## Running

From the repo root:

```sh
sh scripts/test-integration.sh
```

Current container-backed smoke coverage:

- centralized matrix package: `integration/all`
- Google Pub/Sub emulator startup via `testcontainers-go/modules/gcloud/pubsub`
- Kafka startup via `testcontainers-go/modules/kafka`
- NATS startup via `testcontainers-go`
- Redis startup via `testcontainers-go`
- shared distributed-driver scenario suite via `integration/scenario`
- Google Pub/Sub driver delivery via `driver/gcppubsubevents`
- Kafka driver delivery via `driver/kafkaevents`
- NATS driver delivery via `driver/natsevents`
- Redis driver delivery via `driver/redisevents`

To limit the matrix to specific fixtures:

```sh
cd integration
INTEGRATION_DRIVER=gcppubsub go test ./all
INTEGRATION_DRIVER=nats go test ./all
INTEGRATION_DRIVER=kafka go test ./all
INTEGRATION_DRIVER=redis go test ./all
```
