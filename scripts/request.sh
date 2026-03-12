curl -sS -X POST http://localhost:8080/script \
  -H "Content-Type: application/json" \
  -d '{"files":["testdata/alpha.txt","testdata/bravo.txt"]}'
