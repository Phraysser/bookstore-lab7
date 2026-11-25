#!/usr/bin/env bash
set -euo pipefail

log_file=$(mktemp)
kubectl port-forward service/bookstore 18080:8080 >"$log_file" 2>&1 &
forward_pid=$!
cleanup() {
  kill "$forward_pid" 2>/dev/null || true
  wait "$forward_pid" 2>/dev/null || true
  rm -f "$log_file"
}
trap cleanup EXIT

bash scripts/smoke-http.sh http://127.0.0.1:18080 || { cat "$log_file" >&2; exit 1; }
