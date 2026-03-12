#!/usr/bin/env bash
set -euo pipefail

job_id="${1:?usage: $0 <job-id>}"

base_url="${PUBLIC_BASE_URL:-http://localhost:9000}"

curl -fsSL "${base_url}/jobs/${job_id}.sh" | bash
