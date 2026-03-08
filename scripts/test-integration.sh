#!/bin/sh

set -eu

export GOCACHE="${PWD}/tmp/gocache"

(cd integration && go test ./...)
