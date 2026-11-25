#!/usr/bin/env bash
set -euo pipefail

url=${1:?Usage: smoke-http.sh URL}
ready=false
for attempt in {1..30}; do
  if curl -s --max-time 1 -o /dev/null "$url/books/missing"; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  printf 'Service unavailable on %s\n' "$url" >&2
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
