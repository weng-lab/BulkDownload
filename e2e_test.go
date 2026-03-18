package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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

func useTestConfig(t *testing.T, zipTTL, cleanupTick, processingDelay time.Duration) string {
	t.Helper()

	t.Cleanup(func() {
		core.LoadConfig()
	})

	jobsDir := filepath.Join(t.TempDir(), "jobs")
	t.Setenv("JOBS_DIR", jobsDir)
	t.Setenv("SOURCE_ROOT_DIR", "")
	t.Setenv("PUBLIC_BASE_URL", "https://download.mohd.org")
	t.Setenv("DOWNLOAD_ROOT_DIR", "mohd_data")
	t.Setenv("ZIP_TTL", zipTTL.String())
	t.Setenv("CLEANUP_TICK", cleanupTick.String())
	t.Setenv("PROCESSING_DELAY", processingDelay.String())
	core.LoadConfig()

	if err := os.MkdirAll(core.JobsDir, 0o755); err != nil {
		t.Fatalf("create jobs dir: %v", err)
	}

	return jobsDir
}

func TestEndToEndZipLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	useTestConfig(t, 3*time.Second, 500*time.Millisecond, 750*time.Millisecond)

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
	mux.HandleFunc("/tarball", api.HandleCreateTarball(store))
	mux.HandleFunc("/script", api.HandleCreateScript(store))
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

	var created api.JobResponse
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

func TestEndToEndTarballLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	useTestConfig(t, 3*time.Second, 500*time.Millisecond, 750*time.Millisecond)

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
	mux.HandleFunc("/tarball", api.HandleCreateTarball(store))
	mux.HandleFunc("/script", api.HandleCreateScript(store))
	mux.HandleFunc("/status/", api.HandleStatus(store))
	mux.HandleFunc("/download/", api.HandleDownload(store))

	server := httptest.NewServer(mux)
	defer server.Close()

	createResp, err := http.Post(server.URL+"/tarball", "application/json", strings.NewReader(`{"files":["`+alphaPath+`","`+bravoPath+`"]}`))
	if err != nil {
		t.Fatalf("create tarball request: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected create status %d, got %d: %s", http.StatusAccepted, createResp.StatusCode, string(body))
	}

	var created api.JobResponse
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
				t.Fatalf("download tarball: %v", err)
			}
			defer downloadResp.Body.Close()

			if downloadResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(downloadResp.Body)
				t.Fatalf("expected download status %d, got %d: %s", http.StatusOK, downloadResp.StatusCode, string(body))
			}

			archivePath := filepath.Join(tempDir, "downloaded.tar.gz")
			archiveData, err := io.ReadAll(downloadResp.Body)
			if err != nil {
				t.Fatalf("read download response: %v", err)
			}
			if err := os.WriteFile(archivePath, archiveData, 0o644); err != nil {
				t.Fatalf("write downloaded archive: %v", err)
			}

			archiveFile, err := os.Open(archivePath)
			if err != nil {
				t.Fatalf("open downloaded archive: %v", err)
			}
			defer archiveFile.Close()

			gzReader, err := gzip.NewReader(archiveFile)
			if err != nil {
				t.Fatalf("open gzip reader: %v", err)
			}
			defer gzReader.Close()

			tarReader := tar.NewReader(gzReader)
			entryCount := 0
			for {
				_, err := tarReader.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("read tar entry: %v", err)
				}
				entryCount++
			}

			if entryCount != 2 {
				t.Fatalf("expected 2 files in archive, got %d", entryCount)
			}

			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job completion")
}

func TestEndToEndScriptLifecycle(t *testing.T) {
	useTestConfig(t, 3*time.Second, 500*time.Millisecond, 250*time.Millisecond)

	store := core.NewStore()
	core.StartCleanup(store)
	mux := http.NewServeMux()
	mux.HandleFunc("/zip", api.HandleCreateZip(store))
	mux.HandleFunc("/tarball", api.HandleCreateTarball(store))
	mux.HandleFunc("/script", api.HandleCreateScript(store))
	mux.HandleFunc("/status/", api.HandleStatus(store))
	mux.HandleFunc("/download/", api.HandleDownload(store))

	server := httptest.NewServer(mux)
	defer server.Close()

	createResp, err := http.Post(server.URL+"/script", "application/json", strings.NewReader(`{"files":["rna/accession.bigwig","dna/sample.cram"]}`))
	if err != nil {
		t.Fatalf("create script request: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected create status %d, got %d: %s", http.StatusAccepted, createResp.StatusCode, string(body))
	}

	var created api.JobResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

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

		if job.Status == core.StatusDone {
			scriptPath := filepath.Join(core.JobsDir, job.Filename)
			data, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("read generated script: %v", err)
			}
			content := string(data)
			if !strings.Contains(content, "BASE_URL='https://download.mohd.org'") {
				t.Fatalf("expected script to include base URL, got %q", content)
			}
			if !strings.Contains(content, "'rna/accession.bigwig'") {
				t.Fatalf("expected script to include first relative path, got %q", content)
			}
			if !strings.Contains(content, "DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-'mohd_data'}") {
				t.Fatalf("expected script to include download root default, got %q", content)
			}
			if !strings.Contains(content, "MAX_JOBS=${MAX_JOBS:-3}") {
				t.Fatalf("expected script to include max jobs default, got %q", content)
			}
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for script job completion")
}
