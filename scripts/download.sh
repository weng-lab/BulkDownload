#!/usr/bin/env bash
set -euo pipefail

job_id="${1:?usage: $0 <job-id>}"

base_url="${API_BASE_URL:-http://localhost:8080}"

curl -fsSL "${base_url%/}/download/${job_id}" | bash
