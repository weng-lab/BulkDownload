package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCreateDownloadScript_WritesExpectedContent(t *testing.T) {
	t.Parallel()

	const wantScriptPerm os.FileMode = 0o755

	tests := []struct {
		name         string
		baseURL      string
		downloadRoot string
		files        []string
		wantContent  string
		wantErr      bool
	}{
		{
			name:         "writes expected snippets",
			baseURL:      "https://download.mohd.org",
			downloadRoot: "mohd_data",
			files:        []string{"rna/accession.bigwig", "dna/sample.cram"},
			wantContent:  expectedDownloadScript("https://download.mohd.org", "mohd_data", []string{"rna/accession.bigwig", "dna/sample.cram"}),
		},
		{
			name:         "trims trailing slash from base url",
			baseURL:      "https://download.mohd.org/",
			downloadRoot: "mohd_data",
			files:        []string{"rna/accession.bigwig"},
			wantContent:  expectedDownloadScript("https://download.mohd.org", "mohd_data", []string{"rna/accession.bigwig"}),
		},
		{
			name:         "shell quotes file paths",
			baseURL:      "https://download.mohd.org",
			downloadRoot: "mohd_data",
			files:        []string{"rna/it's.bigwig"},
			wantContent:  expectedDownloadScript("https://download.mohd.org", "mohd_data", []string{"rna/it's.bigwig"}),
		},
		{
			name:         "fails for empty file list",
			baseURL:      "https://download.mohd.org",
			downloadRoot: "mohd_data",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scriptPath := filepath.Join(t.TempDir(), "download.sh")
			err := createDownloadScript(scriptPath, tt.baseURL, tt.downloadRoot, tt.files)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("createDownloadScript() error = nil, want non-nil")
				}
				assertFileAbsent(t, scriptPath)
				return
			}
			if err != nil {
				t.Fatalf("createDownloadScript() error = %v", err)
			}

			data, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", scriptPath, err)
			}
			content := string(data)
			if diff := cmp.Diff(tt.wantContent, content); diff != "" {
				t.Errorf("script content mismatch (-want +got):\n%s", diff)
			}

			info, err := os.Stat(scriptPath)
			if err != nil {
				t.Fatalf("Stat(%q) error = %v", scriptPath, err)
			}
			if diff := cmp.Diff(wantScriptPerm, info.Mode().Perm()); diff != "" {
				t.Errorf("script permissions mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessScriptJob_CreatesScriptAndMarksDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		files   []string
		wantErr bool
	}{
		{
			name:  "marks done after creating script",
			files: []string{"rna/accession.bigwig"},
		},
		{
			name:    "fails for empty file list",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fixture := newTestFixture(t)
			if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
			}

			job, err := fixture.manager.CreateScriptJob(tt.files)
			if err != nil {
				t.Fatalf("CreateScriptJob() error = %v", err)
			}

			err = fixture.manager.ProcessScriptJob(job.ID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ProcessScriptJob() error = nil, want non-nil")
				}
				assertFailedArchiveJob(t, fixture.jobs, job.ID)
				assertFileAbsent(t, filepath.Join(fixture.config.JobsDir, job.ID+".sh"))
				return
			}
			if err != nil {
				t.Fatalf("ProcessScriptJob() error = %v", err)
			}

			got, ok := fixture.jobs.Get(job.ID)
			if !ok {
				t.Fatalf("Get(%q) ok = false, want true", job.ID)
			}
			if got.Filename == "" {
				t.Fatalf("processed job filename = %q, want non-empty", got.Filename)
			}

			want := &Job{
				ID:        job.ID,
				Type:      JobTypeScript,
				Status:    StatusDone,
				Progress:  100,
				ExpiresAt: job.ExpiresAt,
				Files:     append([]string(nil), tt.files...),
				Filename:  got.Filename,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("processed job mismatch (-want +got):\n%s", diff)
			}

			scriptPath := filepath.Join(fixture.config.JobsDir, got.Filename)
			if _, err := os.Stat(scriptPath); err != nil {
				t.Fatalf("Stat(%q) error = %v", scriptPath, err)
			}
		})
	}
}

func expectedDownloadScript(baseURL, downloadRoot string, files []string) string {
	quotedFiles := make([]string, 0, len(files))
	for _, file := range files {
		quotedFiles = append(quotedFiles, shellQuote(file))
	}

	filesBlock := ""
	for i, file := range quotedFiles {
		suffix := " \\\n\n"
		if i == len(quotedFiles)-1 {
			suffix = "\n\n"
		}
		filesBlock += fmt.Sprintf("  %s%s", file, suffix)
	}

	return fmt.Sprintf(`#!/usr/bin/env bash
set -uo pipefail

BASE_URL=%s
DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-%s}
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
  local url="${BASE_URL%%/}/$rel_path"

  mkdir -p "$(dirname "$dest")"
  printf 'Downloading %%s\n' "$rel_path"
  if curl -fL -C - --retry 5 --retry-delay 2 --retry-all-errors -o "$dest" "$url"; then
    printf 'Finished %%s\n' "$rel_path"
  else
    printf '%%s\n' "$rel_path" >>"$failures_file"
    printf 'Failed %%s\n' "$rel_path" >&2
  fi
}

for rel_path in \
%sdo
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
    printf '  - %%s\n' "$failed_path" >&2
  done <"$failures_file"
  exit 1
fi

printf '\nAll downloads completed successfully.\n'
`, shellQuote(baseURL), shellQuote(downloadRoot), filesBlock)
}
