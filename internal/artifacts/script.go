package artifacts

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const downloadScriptTemplate = `#!/usr/bin/env bash
set -uo pipefail

BASE_URL={{.BaseURL}}
DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-{{.DownloadRoot}}}
MAX_JOBS=${MAX_JOBS:-{{.MaxJobs}}}

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
{{- range .Files }}
  {{.Value}}{{.LineSuffix}}
{{- end }}
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
`

type scriptTemplateData struct {
	BaseURL      string
	DownloadRoot string
	MaxJobs      int
	Files        []scriptTemplateFile
}

type scriptTemplateFile struct {
	Value      string
	LineSuffix string
}

func CreateDownloadScript(dest, baseURL, downloadRoot string, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files provided")
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create script file: %w", err)
	}
	defer f.Close()

	templateFiles := make([]scriptTemplateFile, 0, len(files))
	for i, file := range files {
		lineSuffix := " \\\n"
		if i == len(files)-1 {
			lineSuffix = "\n"
		}
		templateFiles = append(templateFiles, scriptTemplateFile{
			Value:      shellQuote(file),
			LineSuffix: lineSuffix,
		})
	}

	data := scriptTemplateData{
		BaseURL:      shellQuote(strings.TrimRight(baseURL, "/")),
		DownloadRoot: shellQuote(downloadRoot),
		MaxJobs:      3,
		Files:        templateFiles,
	}

	tmpl, err := template.New("download-script").Parse(downloadScriptTemplate)
	if err != nil {
		return fmt.Errorf("parse script template: %w", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("write script file: %w", err)
	}

	if err := f.Chmod(0o755); err != nil {
		return fmt.Errorf("chmod script file: %w", err)
	}

	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
