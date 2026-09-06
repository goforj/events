module github.com/goforj/events/examples

go 1.26.0

require (
	github.com/goforj/events v0.2.0
	github.com/goforj/events/driver/gcppubsubevents v0.2.0
	github.com/goforj/events/driver/kafkaevents v0.2.0
	github.com/goforj/events/driver/natsevents v0.2.0
	github.com/goforj/events/driver/natsjetstreamevents v0.2.0
	github.com/goforj/events/driver/redisevents v0.2.0
	github.com/goforj/events/driver/snsevents v0.2.0
	github.com/goforj/events/eventscore v0.2.0
	github.com/nats-io/nats.go v1.53.1
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.11.0 // indirect
	cloud.google.com/go/pubsub v1.51.1 // indirect
	cloud.google.com/go/pubsub/v2 v2.6.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.45.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.33.2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.20.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.45.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.50.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.36.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.41.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.48.0 // indirect
	github.com/aws/smithy-go v1.28.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.17 // indirect
	github.com/googleapis/gax-go/v2 v2.23.0 // indirect
	github.com/klauspost/compress v1.18.7 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	github.com/segmentio/kafka-go v0.4.51 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.67.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.56.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/api v0.287.1 // indirect
	google.golang.org/genproto v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260630182238-925bb5da69e7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260630182238-925bb5da69e7 // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/goforj/events => ./..

replace github.com/goforj/events/driver/gcppubsubevents => ../driver/gcppubsubevents

replace github.com/goforj/events/driver/kafkaevents => ../driver/kafkaevents

replace github.com/goforj/events/driver/natsevents => ../driver/natsevents

replace github.com/goforj/events/driver/natsjetstreamevents => ../driver/natsjetstreamevents

replace github.com/goforj/events/driver/redisevents => ../driver/redisevents

replace github.com/goforj/events/driver/snsevents => ../driver/snsevents

replace github.com/goforj/events/eventscore => ../eventscore
