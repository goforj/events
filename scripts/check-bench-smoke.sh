#!/bin/sh

set -eu

export GOCACHE="${PWD}/tmp/gocache"

go test -run '^$' -bench 'Benchmark(PublishNoSubscribers|PublishOneSubscriber|PublishMultipleSubscribers|ResolveTopic|NewRegisteredHandler)$' -benchmem

if [ "${RUN_INTEGRATION:-0}" = "1" ]; then
  (
    cd integration
    go test ./bench -run '^$' -bench 'BenchmarkDistributedPublishRoundTrip' -benchtime=1x -benchmem
  )
fi
