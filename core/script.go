package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -uo pipefail\n\n")
	b.WriteString(fmt.Sprintf("BASE_URL=%s\n", shellQuote(strings.TrimRight(PublicBaseURL, "/"))))
	b.WriteString(fmt.Sprintf("DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-%s}\n", shellQuote(DownloadRootDir)))
	b.WriteString("MAX_JOBS=${MAX_JOBS:-3}\n\n")
	b.WriteString("mkdir -p \"$DOWNLOAD_ROOT\"\n")
	b.WriteString("failures_file=$(mktemp)\n")
	b.WriteString("cleanup() {\n")
	b.WriteString("  rm -f \"$failures_file\"\n")
	b.WriteString("}\n")
	b.WriteString("trap cleanup EXIT\n\n")
	b.WriteString("download_file() {\n")
	b.WriteString("  local rel_path=$1\n")
	b.WriteString("  local dest=\"$DOWNLOAD_ROOT/$rel_path\"\n")
	b.WriteString("  local url=\"${BASE_URL%/}/$rel_path\"\n\n")
	b.WriteString("  mkdir -p \"$(dirname \"$dest\")\"\n")
	b.WriteString("  printf 'Downloading %s\\n' \"$rel_path\"\n")
	b.WriteString("  if curl -fL -C - --retry 5 --retry-delay 2 --retry-all-errors -o \"$dest\" \"$url\"; then\n")
	b.WriteString("    printf 'Finished %s\\n' \"$rel_path\"\n")
	b.WriteString("  else\n")
	b.WriteString("    printf '%s\\n' \"$rel_path\" >>\"$failures_file\"\n")
	b.WriteString("    printf 'Failed %s\\n' \"$rel_path\" >&2\n")
	b.WriteString("  fi\n")
	b.WriteString("}\n\n")
	b.WriteString("for rel_path in \\\n")
	for i, file := range files {
		lineSuffix := " \\\n"
		if i == len(files)-1 {
			lineSuffix = "\n"
		}
		b.WriteString("  ")
		b.WriteString(shellQuote(file))
		b.WriteString(lineSuffix)
	}
	b.WriteString("do\n")
	b.WriteString("  while [ \"$(jobs -rp | wc -l | tr -d ' ')\" -ge \"$MAX_JOBS\" ]; do\n")
	b.WriteString("    wait -n || true\n")
	b.WriteString("  done\n\n")
	b.WriteString("  download_file \"$rel_path\" &\n")
	b.WriteString("done\n\n")
	b.WriteString("while [ \"$(jobs -rp | wc -l | tr -d ' ')\" -gt 0 ]; do\n")
	b.WriteString("  wait -n || true\n")
	b.WriteString("done\n\n")
	b.WriteString("if [ -s \"$failures_file\" ]; then\n")
	b.WriteString("  printf '\\nSome downloads failed:\\n' >&2\n")
	b.WriteString("  while IFS= read -r failed_path; do\n")
	b.WriteString("    printf '  - %s\\n' \"$failed_path\" >&2\n")
	b.WriteString("  done <\"$failures_file\"\n")
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n\n")
	b.WriteString("printf '\\nAll downloads completed successfully.\\n'\n")

	if _, err := f.WriteString(b.String()); err != nil {
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
