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

	"github.com/go-chi/chi/v5"
	"github.com/jair/bulkdownload/api"
	"github.com/jair/bulkdownload/core"
)

type testApp struct {
	config core.Config
	jobs   *core.Jobs
	server *httptest.Server
}

func newTestApp(t *testing.T, config core.Config) testApp {
	t.Helper()

	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("create jobs dir: %v", err)
	}
	if config.SourceRootDir != "" {
		if err := os.MkdirAll(config.SourceRootDir, 0o755); err != nil {
			t.Fatalf("create source root dir: %v", err)
		}
	}

	jobs := core.NewJobs()
	manager := core.NewManager(jobs, config)
	core.StartCleanup(jobs, config.JobsDir, config.CleanupTick)

	mux := chi.NewRouter()
	mux.Post("/zip", api.HandleCreateZip(manager, config))
	mux.Post("/tarball", api.HandleCreateTarball(manager, config))
	mux.Post("/script", api.HandleCreateScript(manager, config))
	mux.Get("/status/{id}", api.HandleStatus(jobs))
	mux.Get("/download/{id}", api.HandleDownload(jobs, config))

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	return testApp{
		config: config,
		jobs:   jobs,
		server: server,
	}
}

func testConfig(t *testing.T) core.Config {
	t.Helper()

	root := t.TempDir()
	return core.Config{
		JobsDir:         filepath.Join(root, "jobs"),
		SourceRootDir:   filepath.Join(root, "source"),
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		Port:            "0",
		JobTTL:          300 * time.Millisecond,
		CleanupTick:     50 * time.Millisecond,
	}
}

func writeAppSourceFile(t *testing.T, root, relPath, contents string) string {
	t.Helper()

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	return path
}

func TestEndToEndZipLifecycle(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")

	created := createJob(t, app.server.URL+"/zip", `{"files":["nested/alpha.txt","nested/bravo.txt"]}`)
	job := waitForDoneStatus(t, app.server.URL, created.ID)

	resp, err := http.Get(app.server.URL + "/download/" + created.ID)
	if err != nil {
		t.Fatalf("download zip: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected download status %d, got %d: %s", http.StatusOK, resp.StatusCode, string(body))
	}

	archivePath := filepath.Join(t.TempDir(), "downloaded.zip")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read zip body: %v", err)
	}
	if err := os.WriteFile(archivePath, body, 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	if len(reader.File) != 2 {
		t.Fatalf("expected 2 files in archive, got %d", len(reader.File))
	}
	if job.Status != core.StatusDone || job.Progress != 100 {
		t.Fatalf("expected completed job, got %#v", job)
	}
}

func TestEndToEndTarballLifecycle(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")

	created := createJob(t, app.server.URL+"/tarball", `{"files":["nested/alpha.txt","nested/bravo.txt"]}`)
	job := waitForDoneStatus(t, app.server.URL, created.ID)

	resp, err := http.Get(app.server.URL + "/download/" + created.ID)
	if err != nil {
		t.Fatalf("download tarball: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected download status %d, got %d: %s", http.StatusOK, resp.StatusCode, string(body))
	}

	archivePath := filepath.Join(t.TempDir(), "downloaded.tar.gz")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read tarball body: %v", err)
	}
	if err := os.WriteFile(archivePath, body, 0o644); err != nil {
		t.Fatalf("write tarball file: %v", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open tarball: %v", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	entryCount := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}
		entryCount++
	}

	if entryCount != 2 {
		t.Fatalf("expected 2 files in tarball, got %d", entryCount)
	}
	if job.Status != core.StatusDone || job.Progress != 100 {
		t.Fatalf("expected completed job, got %#v", job)
	}
}

func TestEndToEndScriptLifecycle(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "rna/accession.bigwig", "rna data")
	writeAppSourceFile(t, config.SourceRootDir, "dna/sample.cram", "dna data")

	created := createJob(t, app.server.URL+"/script", `{"files":["rna/accession.bigwig","dna/sample.cram"]}`)
	job := waitForDoneStatus(t, app.server.URL, created.ID)

	resp, err := http.Get(app.server.URL + "/download/" + created.ID)
	if err != nil {
		t.Fatalf("download script: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected download status %d, got %d: %s", http.StatusOK, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read script body: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, "BASE_URL='https://download.mohd.org'") {
		t.Fatalf("expected script to include base URL, got %q", content)
	}
	if !strings.Contains(content, "'rna/accession.bigwig'") {
		t.Fatalf("expected script to include first relative path, got %q", content)
	}
	if !strings.Contains(content, "DOWNLOAD_ROOT=${DOWNLOAD_ROOT:-'mohd_data'}") {
		t.Fatalf("expected script to include download root, got %q", content)
	}
	if job.Status != core.StatusDone || job.Progress != 100 {
		t.Fatalf("expected completed job, got %#v", job)
	}
}

func TestCleanupRemovesExpiredJobAndArtifact(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.JobTTL = 120 * time.Millisecond
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "rna/accession.bigwig", "rna data")

	created := createJob(t, app.server.URL+"/script", `{"files":["rna/accession.bigwig"]}`)
	job := waitForDoneStatus(t, app.server.URL, created.ID)
	artifactPath := filepath.Join(config.JobsDir, job.Filename)
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("expected artifact to exist before cleanup: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(app.server.URL + "/status/" + created.ID)
		if err != nil {
			t.Fatalf("get status during cleanup wait: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
				t.Fatalf("expected artifact to be removed, stat err = %v", err)
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for cleanup of job %s", created.ID)
}

func createJob(t *testing.T, url, body string) api.JobResponse {
	t.Helper()

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create job request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected create status %d, got %d: %s", http.StatusAccepted, resp.StatusCode, string(data))
	}

	var created api.JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	return created
}

func waitForDoneStatus(t *testing.T, baseURL, id string) core.Job {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/status/" + id)
		if err != nil {
			t.Fatalf("get status: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("expected status endpoint to return 200, got %d: %s", resp.StatusCode, string(data))
		}

		var job core.Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			resp.Body.Close()
			t.Fatalf("decode status response: %v", err)
		}
		resp.Body.Close()

		if job.Status == core.StatusDone {
			return job
		}
		if job.Status == core.StatusFailed {
			t.Fatalf("job %s failed: %s", id, job.Error)
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job %s to complete", id)
	return core.Job{}
}
