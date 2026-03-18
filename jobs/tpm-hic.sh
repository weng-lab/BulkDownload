#!/usr/bin/env bash
set -uo pipefail

BASE_URL='http://localhost:9000'
DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-'downloads'}
MAX_JOBS=${MAX_JOBS:-3}

mkdir -p "$DOWNLOAD_ROOT"
failures_file=$(mktemp)
cleanup() {
  rm -f "$failures_file"
}
trap cleanup EXIT

download_file() {
  local rel_path=$1
  local dest="$DOWNLOAD_ROOT/$rel_path"
  local url="${BASE_URL%/}/$rel_path"

  mkdir -p "$(dirname "$dest")"
  printf 'Downloading %s\n' "$rel_path"
  if curl -fL -C - --retry 5 --retry-delay 2 --retry-all-errors -o "$dest" "$url"; then
    printf 'Finished %s\n' "$rel_path"
  else
    printf '%s\n' "$rel_path" >>"$failures_file"
    printf 'Failed %s\n' "$rel_path" >&2
  fi
}

for rel_path in \
  'testdata/alpha.txt' \

  'testdata/bravo.txt'

do
  while [ "$(jobs -rp | wc -l | tr -d ' ')" -ge "$MAX_JOBS" ]; do
    wait -n || true
  done

  download_file "$rel_path" &
done

while [ "$(jobs -rp | wc -l | tr -d ' ')" -gt 0 ]; do
  wait -n || true
done

if [ -s "$failures_file" ]; then
  printf '\nSome downloads failed:\n' >&2
  while IFS= read -r failed_path; do
    printf '  - %s\n' "$failed_path" >&2
  done <"$failures_file"
  exit 1
fi

printf '\nAll downloads completed successfully.\n'
