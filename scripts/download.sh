#!/usr/bin/env bash
set -euo pipefail

api_url="${API_URL:-http://localhost:8080}"
job_id="${1:?usage: $0 <job-id>}"

mkdir -p ./downloads

curl -fsSL -OJ "${api_url}/download/${job_id}" --output-dir ./downloads
