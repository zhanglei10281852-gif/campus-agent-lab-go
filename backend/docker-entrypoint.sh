#!/bin/sh
set -eu

if [ -n "${BOOTSTRAP_PASSWORD:-}" ]; then
  /campuslab-seed
fi
exec /campuslab-server
