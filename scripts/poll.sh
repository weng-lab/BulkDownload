api_url="${API_URL:-http://localhost:8080}"
job_id="${1:-$(jq -r '.id')}"

curl -sS "${api_url}/status/${job_id}"
