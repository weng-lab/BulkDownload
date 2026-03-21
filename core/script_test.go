package core

import (
	"os"
	"path/filepath"
	"strings"
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
		checks       []string
		wantErr      bool
	}{
		{
			name:         "writes expected snippets",
			baseURL:      "https://download.mohd.org",
			downloadRoot: "mohd_data",
			files:        []string{"rna/accession.bigwig", "dna/sample.cram"},
			checks: []string{
				"#!/usr/bin/env bash",
				"set -uo pipefail",
				"BASE_URL='https://download.mohd.org'",
				"DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-'mohd_data'}",
				"MAX_JOBS=${MAX_JOBS:-3}",
				"curl -fL -C - --retry 5 --retry-delay 2 --retry-all-errors -o \"$dest\" \"$url\"",
				"'rna/accession.bigwig'",
				"'dna/sample.cram'",
			},
		},
		{
			name:         "trims trailing slash from base url",
			baseURL:      "https://download.mohd.org/",
			downloadRoot: "mohd_data",
			files:        []string{"rna/accession.bigwig"},
			checks: []string{
				"BASE_URL='https://download.mohd.org'",
			},
		},
		{
			name:         "shell quotes file paths",
			baseURL:      "https://download.mohd.org",
			downloadRoot: "mohd_data",
			files:        []string{"rna/it's.bigwig"},
			checks: []string{
				"'rna/it'\"'\"'s.bigwig'",
			},
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

			for _, check := range tt.checks {
				if !strings.Contains(content, check) {
					t.Errorf("script missing %q", check)
				}
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
			manager, jobs, config := testManager(t)
			if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
			}

			job, err := manager.CreateScriptJob(tt.files)
			if err != nil {
				t.Fatalf("CreateScriptJob() error = %v", err)
			}

			err = manager.ProcessScriptJob(job.ID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ProcessScriptJob() error = nil, want non-nil")
				}
				assertFailedArchiveJob(t, jobs, job.ID)
				assertFileAbsent(t, filepath.Join(config.JobsDir, job.ID+".sh"))
				return
			}
			if err != nil {
				t.Fatalf("ProcessScriptJob() error = %v", err)
			}

			got, ok := jobs.Get(job.ID)
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

			scriptPath := filepath.Join(config.JobsDir, got.Filename)
			if _, err := os.Stat(scriptPath); err != nil {
				t.Fatalf("Stat(%q) error = %v", scriptPath, err)
			}
		})
	}
}
