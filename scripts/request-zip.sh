#!/usr/bin/env bash
set -euo pipefail

api_url="${API_URL:-http://localhost:8080}"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_root="$(cd -- "${script_dir}/.." && pwd -P)"
file_one="${1:-${repo_root}/testdata/alpha.txt}"
file_two="${2:-${repo_root}/testdata/bravo.txt}"

curl -sS -X POST "${api_url}/zip" \
  -H "Content-Type: application/json" \
  -d '{"files":["'"${file_one}"'","'"${file_two}"'"]}'
