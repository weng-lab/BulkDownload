#!/usr/bin/env bash
set -euo pipefail

api_url="${API_URL:-http://localhost:8080}"
file_one="${1:-testdata/alpha.txt}"
file_two="${2:-testdata/bravo.txt}"

curl -sS -X POST "${api_url}/jobs" \
  -H "Content-Type: application/json" \
  -d '{"type":"tarball","files":["'"${file_one}"'","'"${file_two}"'"]}'
