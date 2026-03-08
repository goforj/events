#!/bin/sh

set -eu

export GOCACHE="${GOCACHE:-/tmp/events-gocache}"
export RUN_INTEGRATION=1

if [ "${BENCH_INTEGRATION_TIME:-}" = "" ]; then
  export BENCH_INTEGRATION_TIME=1s
fi

go test -tags=benchrender ./docs/bench -run TestRenderBenchmarks -count=1
