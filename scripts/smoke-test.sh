#!/usr/bin/env bash
set -euo pipefail

base_url="${1:?usage: smoke-test.sh https://tenant-a.example.click}"

curl -fsS "$base_url/healthz"
item_id="$(curl -fsS -X POST "$base_url/items" \
  -H 'content-type: application/json' \
  -d '{"name":"Demo launch readiness"}' | jq -r '.id')"

curl -fsS -X POST "$base_url/items/$item_id/vote" \
  -H 'content-type: application/json' \
  -d '{"voter_id":"demo-user-1","score":5,"comment":"works"}'

curl -fsS "$base_url/results"
curl -fsS -X POST "$base_url/exports"
