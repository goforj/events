#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

export GOCACHE="${GOCACHE:-/tmp/gocache}"
export GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}"

bash scripts/check-published-modules.sh

mapfile -t published_modules < scripts/published-modules.txt
mapfile -t workspace_modules < <(
  GOWORK=off go work edit -json go.work |
    awk -F'"' '/"DiskPath":/ { path=$4; sub(/^\.[/]/, "", path); if (path == "") path="."; print path }'
)

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
result=0

for index in "${!workspace_modules[@]}"; do
  dir="${workspace_modules[$index]}"
  check_mod="$tmpdir/module-$index.mod"
  check_sum="$tmpdir/module-$index.sum"
  cp "$dir/go.mod" "$check_mod"
  if [[ -f "$dir/go.sum" ]]; then
    cp "$dir/go.sum" "$check_sum"
  fi

  current_path="$(awk '$1 == "module" { print $2; exit }' "$dir/go.mod")"
  for published_dir in "${published_modules[@]}"; do
    if [[ "$published_dir" == "." ]]; then
      sibling_path="github.com/goforj/events"
      sibling_dir="$root"
    else
      sibling_path="github.com/goforj/events/$published_dir"
      sibling_dir="$root/$published_dir"
    fi

    if [[ "$sibling_path" != "$current_path" ]]; then
      go mod edit -modfile="$check_mod" -replace="$sibling_path=$sibling_dir"
    fi
  done

  echo "tidy: $dir"
  if ! (cd "$dir" && GOWORK=off go mod tidy -modfile="$check_mod" -diff); then
    result=1
  fi
done

exit "$result"
