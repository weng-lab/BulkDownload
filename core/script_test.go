package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateDownloadScriptWritesExpectedContent(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 0)

	scriptPath := filepath.Join(t.TempDir(), "download.sh")
	if err := createDownloadScript(scriptPath, []string{"rna/accession.bigwig", "dna/sample.cram"}); err != nil {
		t.Fatalf("create download script: %v", err)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
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
			t.Fatalf("expected script to contain %q, got %q", check, content)
		}
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("expected script permissions 755, got %o", info.Mode().Perm())
	}
}

func TestProcessScriptJobCreatesScriptAndMarksDone(t *testing.T) {
	useTestRuntime(t, 3*time.Second, 5*time.Minute, 100*time.Millisecond)

	store := NewStore()
	job, err := store.CreateJob([]string{"rna/accession.bigwig"})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	go ProcessScriptJob(store, job)

	waitFor(t, 500*time.Millisecond, 25*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusProcessing
	}, "script job to reach processing")

	waitFor(t, 2*time.Second, 50*time.Millisecond, func() bool {
		got, ok := store.Get(job.ID)
		return ok && got.Status == StatusDone && got.Filename != ""
	}, "script job to reach done")

	scriptPath := filepath.Join(JobsDir, job.Filename)
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected script file to exist: %v", err)
	}
}
