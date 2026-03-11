package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jair/bulkdownload/api"
	"github.com/jair/bulkdownload/core"
)

func TestEndToEndZipLifecycle(t *testing.T) {
	prevZipTTL := core.ZipTTL
	prevCleanupTick := core.CleanupTick
	prevProcessingDelay := core.ProcessingDelay
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	tempDir := t.TempDir()

	core.ZipTTL = 3 * time.Second
	core.CleanupTick = 500 * time.Millisecond
	core.ProcessingDelay = 750 * time.Millisecond

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	if err := os.MkdirAll(core.OutputDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	t.Cleanup(func() {
		core.ZipTTL = prevZipTTL
		core.CleanupTick = prevCleanupTick
		core.ProcessingDelay = prevProcessingDelay
		_ = os.Chdir(prevWD)
	})

	alphaPath := filepath.Join(tempDir, "alpha.txt")
	bravoPath := filepath.Join(tempDir, "bravo.txt")
	if err := os.WriteFile(alphaPath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write alpha file: %v", err)
	}
	if err := os.WriteFile(bravoPath, []byte("bravo contents"), 0o644); err != nil {
		t.Fatalf("write bravo file: %v", err)
	}

	store := core.NewStore()
	core.StartCleanup(store)
	mux := http.NewServeMux()
	mux.HandleFunc("/zip", api.HandleCreateZip(store))
	mux.HandleFunc("/status/", api.HandleStatus(store))
	mux.HandleFunc("/download/", api.HandleDownload(store))

	server := httptest.NewServer(mux)
	defer server.Close()

	createResp, err := http.Post(server.URL+"/zip", "application/json", strings.NewReader(`{"files":["`+alphaPath+`","`+bravoPath+`"]}`))
	if err != nil {
		t.Fatalf("create zip request: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected create status %d, got %d: %s", http.StatusAccepted, createResp.StatusCode, string(body))
	}

	var created api.ZipResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	observedIntermediate := false
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		statusResp, err := http.Get(server.URL + "/status/" + created.ID)
		if err != nil {
			t.Fatalf("get status: %v", err)
		}

		var job core.Job
		if err := json.NewDecoder(statusResp.Body).Decode(&job); err != nil {
			statusResp.Body.Close()
			t.Fatalf("decode status response: %v", err)
		}
		statusResp.Body.Close()

		if job.Status == core.StatusPending || job.Status == core.StatusProcessing {
			observedIntermediate = true
		}
		if job.Status == core.StatusDone {
			if !observedIntermediate {
				t.Fatalf("expected to observe pending or processing before done")
			}

			downloadResp, err := http.Get(server.URL + "/download/" + created.ID)
			if err != nil {
				t.Fatalf("download zip: %v", err)
			}
			defer downloadResp.Body.Close()

			if downloadResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(downloadResp.Body)
				t.Fatalf("expected download status %d, got %d: %s", http.StatusOK, downloadResp.StatusCode, string(body))
			}

			archivePath := filepath.Join(tempDir, "downloaded.zip")
			archiveData, err := io.ReadAll(downloadResp.Body)
			if err != nil {
				t.Fatalf("read download response: %v", err)
			}
			if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
				t.Fatalf("write downloaded archive: %v", err)
			}

			reader, err := zip.OpenReader(archivePath)
			if err != nil {
				t.Fatalf("open downloaded archive: %v", err)
			}
			defer reader.Close()

			if len(reader.File) != 2 {
				t.Fatalf("expected 2 files in archive, got %d", len(reader.File))
			}

			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job completion")
}
