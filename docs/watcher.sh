#!/bin/sh

set -eu

cd "$(dirname "$0")"

while true
do
  ../scripts/update-docs.sh
  sleep 2
done
