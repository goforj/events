#!/bin/sh

set -eu

export GOCACHE="${PWD}/tmp/gocache"

tmp_dir="${PWD}/tmp/coverage"
mkdir -p "${tmp_dir}"

run_cover() {
  dir="$1"
  out="$2"
  pattern="${3:-./...}"
  (cd "${dir}" && go test ${pattern} -coverprofile="${out}")
}

run_integration_cover() {
  out="$1"
  coverpkg="github.com/goforj/events,github.com/goforj/events/driver/gcppubsubevents,github.com/goforj/events/driver/kafkaevents,github.com/goforj/events/driver/natsevents,github.com/goforj/events/driver/redisevents"
  (cd integration && go test ./... -coverpkg="${coverpkg}" -coverprofile="${out}")
}

run_cover "." "${tmp_dir}/root.out"
run_cover "eventscore" "${tmp_dir}/eventscore.out"
run_cover "eventstest" "${tmp_dir}/eventstest.out"
run_cover "eventsfake" "${tmp_dir}/eventsfake.out"
run_cover "examples" "${tmp_dir}/examples.out" "."
run_cover "docs" "${tmp_dir}/docs.out"
run_cover "driver/gcppubsubevents" "${tmp_dir}/gcppubsubevents.out"
run_cover "driver/kafkaevents" "${tmp_dir}/kafkaevents.out"
run_cover "driver/natsevents" "${tmp_dir}/natsevents.out"
run_cover "driver/redisevents" "${tmp_dir}/redisevents.out"

if [ "${RUN_INTEGRATION:-0}" = "1" ]; then
  run_integration_cover "${tmp_dir}/integration.out"
fi

echo "mode: set" > coverage.out
for profile in \
  "${tmp_dir}/root.out" \
  "${tmp_dir}/eventscore.out" \
  "${tmp_dir}/eventstest.out" \
  "${tmp_dir}/eventsfake.out" \
  "${tmp_dir}/examples.out" \
  "${tmp_dir}/docs.out" \
  "${tmp_dir}/gcppubsubevents.out" \
  "${tmp_dir}/kafkaevents.out" \
  "${tmp_dir}/natsevents.out" \
  "${tmp_dir}/redisevents.out" \
  "${tmp_dir}/integration.out"
do
  if [ -f "${profile}" ]; then
    tail -n +2 "${profile}" >> coverage.out
  fi
done

go tool cover -func=coverage.out
