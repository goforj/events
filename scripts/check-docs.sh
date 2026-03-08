#!/bin/sh

set -eu

export GOCACHE="${GOCACHE:-/tmp/events-gocache}"

(cd docs && go run ./readme/main.go)
(cd docs && go run ./examplegen/main.go)
(cd docs && BENCH_RENDER_ONLY=1 go test -tags=benchrender ./bench -run TestRenderBenchmarks -count=1)
