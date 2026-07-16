#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

manifest="scripts/published-modules.txt"
expected_published=(
  "eventscore"
  "."
  "eventstest"
  "eventsfake"
  "driver/gcppubsubevents"
  "driver/kafkaevents"
  "driver/natsevents"
  "driver/natsjetstreamevents"
  "driver/redisevents"
  "driver/snsevents"
)
support_modules=(
  "docs"
  "examples"
  "integration"
)

if [[ ! -f "$manifest" ]]; then
  echo "missing published module manifest: $manifest" >&2
  exit 1
fi

mapfile -t published_modules < "$manifest"
status=0

if [[ ${#published_modules[@]} -ne ${#expected_published[@]} ]]; then
  echo "published module manifest has the wrong number of entries" >&2
  status=1
else
  for index in "${!expected_published[@]}"; do
    if [[ "${published_modules[$index]}" != "${expected_published[$index]}" ]]; then
      echo "published module manifest entry $((index + 1)) must be ${expected_published[$index]}" >&2
      status=1
    fi
  done
fi

canonical_version="$(awk '
  $1 == "require" && $2 == "github.com/goforj/events/eventscore" { print $3 }
  $1 == "github.com/goforj/events/eventscore" && $2 ~ /^v/ { print $2 }
' go.mod)"
canonical_valid=1

if [[ ! "$canonical_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]; then
  echo "root go.mod must require eventscore at a release version" >&2
  status=1
  canonical_valid=0
fi

if [[ "$canonical_version" == "v0.0.0" ]]; then
  echo "root go.mod must not use the unreleased sibling placeholder v0.0.0" >&2
  status=1
  canonical_valid=0
fi

if [[ $# -gt 1 ]]; then
  echo "usage: scripts/check-published-modules.sh [release-version]" >&2
  exit 1
fi

if [[ $# -eq 1 ]] && [[ "$canonical_valid" -eq 1 ]] && [[ "$1" != "$canonical_version" ]]; then
  echo "requested release $1 does not match sibling requirement $canonical_version" >&2
  status=1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

for dir in "${published_modules[@]}"; do
  modfile="$dir/go.mod"
  if [[ ! -f "$modfile" ]]; then
    echo "missing published module file: $modfile" >&2
    status=1
    continue
  fi

  if [[ "$dir" == "." ]]; then
    expected_path="github.com/goforj/events"
  else
    expected_path="github.com/goforj/events/$dir"
  fi

  actual_path="$(awk '$1 == "module" { print $2; exit }' "$modfile")"
  if [[ "$actual_path" != "$expected_path" ]]; then
    echo "$modfile declares $actual_path; expected $expected_path" >&2
    status=1
  fi

  if grep -Eq '^[[:space:]]*replace([[:space:]]|\()' "$modfile"; then
    echo "published module contains a replace directive: $modfile" >&2
    status=1
  fi

  if ! awk -v version="$canonical_version" -v enforce="$canonical_valid" '
    function sibling(path) {
      return path == "github.com/goforj/events" || path ~ /^github[.]com\/goforj\/events\//
    }
    function check(path, required) {
      if (required == "v0.0.0") {
        print FILENAME ":" FNR ": sibling requirement " path " must not use v0.0.0" > "/dev/stderr"
        invalid = 1
      } else if (enforce && required != version) {
        print FILENAME ":" FNR ": sibling requirement " path " must use " version > "/dev/stderr"
        invalid = 1
      }
    }
    $1 == "require" && sibling($2) {
      check($2, $3)
    }
    sibling($1) && $2 ~ /^v/ {
      check($1, $2)
    }
    END { exit invalid }
  ' "$modfile"; then
    status=1
  fi
done

printf '%s\n' "${published_modules[@]}" "${support_modules[@]}" | sort > "$tmpdir/classified"
find . -name go.mod -type f \
  -not -path './.git/*' \
  -not -path './*/vendor/*' \
  -exec dirname {} \; | sed 's#^\./##; s#^$#.#' | sort > "$tmpdir/discovered"

if ! diff -u "$tmpdir/classified" "$tmpdir/discovered"; then
  echo "every go.mod must be classified as published or support-only" >&2
  status=1
fi

GOWORK=off go work edit -json go.work |
  awk -F'"' '/"DiskPath":/ { path=$4; sub(/^\.[/]/, "", path); if (path == "") path="."; print path }' |
  sort > "$tmpdir/workspace"

if ! diff -u "$tmpdir/classified" "$tmpdir/workspace"; then
  echo "go.work must contain every classified module exactly once" >&2
  status=1
fi

workspace_replace_count="$(grep -Ec '^replace[[:space:]]' go.work || true)"
if [[ "$workspace_replace_count" -ne ${#published_modules[@]} ]]; then
  echo "go.work must contain one version-specific replacement per published module" >&2
  status=1
fi

if [[ "$canonical_valid" -eq 1 ]]; then
  for dir in "${published_modules[@]}"; do
    if [[ "$dir" == "." ]]; then
      module_path="github.com/goforj/events"
      workspace_path="."
    else
      module_path="github.com/goforj/events/$dir"
      workspace_path="./$dir"
    fi

    expected_replace="replace $module_path $canonical_version => $workspace_path"
    if ! grep -Fqx "$expected_replace" go.work; then
      echo "go.work is missing local release binding: $expected_replace" >&2
      status=1
    fi
  done
fi

exit "$status"
