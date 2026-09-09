#!/bin/sh
# Resolve paths independently of cron's working directory.
set -eu
cd "$(dirname "$0")/.."
set -a
. ./.env
set +a
exec ./remo-monitor "$@"
