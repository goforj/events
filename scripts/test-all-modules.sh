#!/bin/sh

set -eu

export GOCACHE="${GOCACHE:-/tmp/gocache}"
export GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}"

run_tests() {
  dir="$1"
  pkgs="$(cd "${dir}" && go list ./...)"
  if [ -n "${pkgs}" ]; then
    # GO_TEST_FLAGS intentionally supports a space-delimited flag list for CI
    # modes such as "-race -count=1".
    # shellcheck disable=SC2086
    (cd "${dir}" && go test ${GO_TEST_FLAGS:-} ./...)
  fi
}

run_tests "."
run_tests "eventscore"
run_tests "eventstest"
run_tests "eventsfake"
run_tests "docs"
run_tests "examples"
run_tests "driver/gcppubsubevents"
run_tests "driver/kafkaevents"
run_tests "driver/natsjetstreamevents"
run_tests "driver/natsevents"
run_tests "driver/redisevents"
run_tests "driver/snsevents"

if [ "${RUN_INTEGRATION:-0}" = "1" ]; then
  run_tests "integration"
fi
