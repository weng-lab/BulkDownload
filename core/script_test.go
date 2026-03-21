package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDownloadScriptWritesExpectedContent(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "download.sh")
	if err := createDownloadScript(scriptPath, "https://download.mohd.org", "mohd_data", []string{"rna/accession.bigwig", "dna/sample.cram"}); err != nil {
		t.Fatalf("createDownloadScript() error = %v", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	checks := []string{
		"#!/usr/bin/env bash",
		"set -uo pipefail",
		"BASE_URL='https://download.mohd.org'",
		"DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-'mohd_data'}",
		"MAX_JOBS=${MAX_JOBS:-3}",
		"curl -fL -C - --retry 5 --retry-delay 2 --retry-all-errors -o \"$dest\" \"$url\"",
		"'rna/accession.bigwig'",
		"'dna/sample.cram'",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Fatalf("script missing %q", check)
		}
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("permissions = %o, want 755", info.Mode().Perm())
	}
}

func TestManagerProcessScriptJobCreatesScriptAndMarksDone(t *testing.T) {
	manager, jobs, config := testManager(t)
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	job, err := manager.CreateScriptJob([]string{"rna/accession.bigwig"})
	if err != nil {
		t.Fatalf("CreateScriptJob() error = %v", err)
	}

	if err := manager.ProcessScriptJob(job.ID); err != nil {
		t.Fatalf("ProcessScriptJob() error = %v", err)
	}

	got, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if got.Status != StatusDone || got.Progress != 100 || got.Filename == "" {
		t.Fatalf("processed job = %#v, want done job with filename", got)
	}

	scriptPath := filepath.Join(config.JobsDir, got.Filename)
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}
