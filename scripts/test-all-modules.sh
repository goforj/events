#!/bin/sh

set -eu

export GOCACHE="${PWD}/tmp/gocache"

run_tests() {
  dir="$1"
  pkgs="$(cd "${dir}" && go list ./... 2>/dev/null || true)"
  if [ -n "${pkgs}" ]; then
    (cd "${dir}" && go test ./...)
  fi
}

go test ./...
run_tests "eventscore"
run_tests "eventstest"
run_tests "eventsfake"
run_tests "examples"
run_tests "docs"
run_tests "driver/gcppubsubevents"
run_tests "driver/kafkaevents"
run_tests "driver/natsevents"
run_tests "driver/redisevents"

if [ "${RUN_INTEGRATION:-0}" = "1" ]; then
  run_tests "integration"
fi
