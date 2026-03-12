package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
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

var downloadScript = template.Must(template.New("download-script").Parse(downloadScriptTemplate))

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

func ProcessScriptJob(store *Store, job *Job) {
	store.SetStatus(job.ID, StatusProcessing)

	log.Printf("script job %s processing started", job.ID)
	log.Printf("script job %s delaying creation for %s", job.ID, ProcessingDelay)
	time.Sleep(ProcessingDelay)
	log.Printf("script job %s delay finished, creating script", job.ID)

	filename := job.ID + ".sh"
	outPath := filepath.Join(JobsDir, filename)

	if err := createDownloadScript(outPath, job.Files); err != nil {
		log.Printf("script generation failed for job %s: %v", job.ID, err)
		if removeErr := os.Remove(outPath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("cleanup failed script for job %s: %v", job.ID, removeErr)
		}
		store.SetFailed(job.ID, err)
		return
	}

	store.SetDone(job.ID, filename)
	log.Printf("script job %s complete: %s", job.ID, outPath)
}

func createDownloadScript(dest string, files []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create script file: %w", err)
	}
	defer f.Close()

	data := newScriptTemplateData(files)
	if err := downloadScript.Execute(f, data); err != nil {
		return fmt.Errorf("write script file: %w", err)
	}

	if err := f.Chmod(0o755); err != nil {
		return fmt.Errorf("chmod script file: %w", err)
	}

	return nil
}

func newScriptTemplateData(files []string) scriptTemplateData {
	quotedFiles := make([]string, 0, len(files))
	for _, file := range files {
		quotedFiles = append(quotedFiles, shellQuote(file))
	}

	templateFiles := make([]scriptTemplateFile, 0, len(quotedFiles))
	for i, file := range quotedFiles {
		lineSuffix := " \\\n"
		if i == len(quotedFiles)-1 {
			lineSuffix = "\n"
		}
		templateFiles = append(templateFiles, scriptTemplateFile{
			Value:      file,
			LineSuffix: lineSuffix,
		})
	}

	return scriptTemplateData{
		BaseURL:      shellQuote(strings.TrimRight(PublicBaseURL, "/")),
		DownloadRoot: shellQuote(DownloadRootDir),
		MaxJobs:      3,
		Files:        templateFiles,
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
