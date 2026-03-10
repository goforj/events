#!/bin/sh

set -eu

version="${1:-}"

if [ -z "${version}" ]; then
  echo "usage: $0 vX.Y.Z" >&2
  exit 1
fi

case "${version}" in
  v*)
    ;;
  *)
    echo "version must start with v" >&2
    exit 1
    ;;
esac

tag_if_missing() {
  tag="$1"
  if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null 2>&1; then
    echo "tag exists: ${tag}"
    return
  fi
  git tag "${tag}"
  echo "created tag: ${tag}"
}

tag_if_missing "${version}"
tag_if_missing "eventscore/${version}"
tag_if_missing "eventstest/${version}"
tag_if_missing "eventsfake/${version}"
tag_if_missing "examples/${version}"
tag_if_missing "docs/${version}"
tag_if_missing "driver/gcppubsubevents/${version}"
tag_if_missing "driver/kafkaevents/${version}"
tag_if_missing "driver/natsevents/${version}"
tag_if_missing "driver/redisevents/${version}"
tag_if_missing "driver/snsevents/${version}"
tag_if_missing "integration/${version}"
