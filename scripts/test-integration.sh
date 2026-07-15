#!/bin/sh

set -eu

# GO_TEST_FLAGS intentionally supports a space-delimited flag list for CI
# modes such as "-race -count=1".
# shellcheck disable=SC2086
(cd integration && go test ${GO_TEST_FLAGS:-} ./root ./all)
