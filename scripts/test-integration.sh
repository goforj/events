#!/bin/sh

set -eu

(cd integration && go test ./root ./all)
