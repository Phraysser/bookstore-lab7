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

url=http://127.0.0.1:18080
ready=false
for attempt in {1..30}; do
  if curl -s --max-time 1 -o /dev/null "$url/books/missing"; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  cat "$log_file" >&2
  exit 1
fi

created=$(curl -fsS -X POST "$url/books" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Go in Action","author":"William Kennedy","price":39.9,"stock":3}')
printf 'POST /books: %s\n' "$created"
id=$(jq -er '.id | select(type == "string" and length > 0)' <<< "$created")
code=$(curl -fsS -o /dev/null -w '%{http_code}' -X PATCH "$url/books/$id/stock" \
  -H 'Content-Type: application/json' -d '{"delta":-3}')
test "$code" = 204
result=$(curl -fsS "$url/books/$id")
printf 'GET /books/%s: %s\n' "$id" "$result"
jq -e '.stock == 0 and .in_stock == false' <<< "$result" >/dev/null
