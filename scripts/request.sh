api_url="${API_URL:-http://localhost:8080}"

curl -sS -X POST "${api_url}/jobs" \
  -H "Content-Type: application/json" \
  -d '{"type":"script","files":["testdata/alpha.txt","testdata/bravo.txt"]}'
