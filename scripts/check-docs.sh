#!/bin/sh

set -eu

export GOCACHE="${PWD}/tmp/gocache"

(cd docs && go run ./readme)
(cd docs && go run ./examplegen)
(cd docs && BENCH_RENDER_ONLY=1 go test -tags=benchrender ./bench -run TestRenderBenchmarks -count=1)
