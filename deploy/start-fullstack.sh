#!/bin/sh
set -eu

if [ -n "${BOOTSTRAP_PASSWORD:-}" ]; then
  campuslab-seed
fi
campuslab-server &
server_pid=$!

shutdown() {
  kill -TERM "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
}

trap shutdown INT TERM EXIT
nginx -g 'daemon off;' &
nginx_pid=$!
wait "$nginx_pid"
