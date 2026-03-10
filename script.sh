#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
DOWNLOAD_DIR="${DOWNLOAD_DIR:-./downloads}"
REQUEST_BODY='{"files":["testdata/alpha.txt","testdata/bravo.txt"]}'

mkdir -p "$DOWNLOAD_DIR"

response_file=$(mktemp)
http_status=$(curl -sS -o "$response_file" -w "%{http_code}" -X POST "$BASE_URL/zip" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_BODY")

response=$(cat "$response_file")
rm -f "$response_file"

echo "Create response: $response"

if [ "$http_status" -ge 400 ]; then
  echo "Request failed with HTTP $http_status"
  exit 1
fi

job_id=$(python -c 'import json,sys; print(json.loads(sys.argv[1])["id"])' "$response")

echo "Job ID: $job_id"
echo "Polling..."

while true; do
  status_json=$(curl -sS "$BASE_URL/status/$job_id")
  status=$(python -c 'import json,sys; print(json.loads(sys.argv[1])["status"])' "$status_json")

  echo "Status: $status"

  if [ "$status" = "done" ]; then
    output_path="$DOWNLOAD_DIR/$job_id.zip"
    echo "Downloading zip to $output_path..."
    curl -sS -o "$output_path" "$BASE_URL/download/$job_id"
    echo "Done"
    break
  fi

  if [ "$status" = "failed" ]; then
    python -c 'import json,sys; d=json.loads(sys.argv[1]); print(d.get("error","unknown error"))' "$status_json"
    exit 1
  fi

  sleep 1
done
