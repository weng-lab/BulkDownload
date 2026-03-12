#!/usr/bin/env bash
set -euo pipefail

job_id="${1:?usage: $0 <job-id>}"

curl -fsSL "http://localhost:9000/jobs/${job_id}.sh" | bash
