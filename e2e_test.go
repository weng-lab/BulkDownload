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
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
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
	stopCleanup := core.StartCleanup(jobs, config.JobsDir, config.CleanupTick)
	t.Cleanup(stopCleanup)

	mux := chi.NewRouter()
	mux.Post("/jobs", api.HandleCreateJob(manager, config))
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

	body := `{"type":"zip","files":["nested/alpha.txt","nested/bravo.txt"]}`
	created := createJob(t, app.server.URL+"/jobs", body)
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
	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read zip body: %v", err)
	}
	if err := os.WriteFile(archivePath, bodyData, 0o644); err != nil {
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

func TestEndToEndZipDownloadContract(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")

	body := `{"type":"zip","files":["nested/alpha.txt","nested/bravo.txt"]}`
	created := createJob(t, app.server.URL+"/jobs", body)
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

	if got := resp.Header.Get("Content-Disposition"); !strings.HasSuffix(got, ".zip\"") {
		t.Fatalf("expected zip attachment, got %q", got)
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read zip body: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "downloaded.zip")
	if err := os.WriteFile(archivePath, bodyData, 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	var gotNames []string
	for _, file := range reader.File {
		gotNames = append(gotNames, file.Name)
	}
	sort.Strings(gotNames)

	wantNames := []string{"nested/alpha.txt", "nested/bravo.txt"}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Fatalf("downloaded zip entries mismatch (-want +got):\n%s", diff)
	}

	if job.Status != core.StatusDone || job.Progress != 100 {
		t.Fatalf("expected completed job, got %#v", job)
	}
}

func TestCreateZipRejectsAbsolutePaths(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)
	alphaPath := writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	bravoPath := writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")

	body := `{"type":"zip","files":["` + alphaPath + `","` + bravoPath + `"]}`
	resp, err := http.Post(app.server.URL+"/jobs", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create zip job with absolute paths: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected create status %d, got %d: %s", http.StatusBadRequest, resp.StatusCode, string(body))
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read error response: %v", err)
	}
	want := "absolute paths are not allowed: " + alphaPath + "\n"
	if string(bodyData) != want {
		t.Fatalf("expected body %q, got %q", want, string(bodyData))
	}
}

func TestEndToEndTarballLifecycle(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")

	body := `{"type":"tarball","files":["nested/alpha.txt","nested/bravo.txt"]}`
	created := createJob(t, app.server.URL+"/jobs", body)
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
	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read tarball body: %v", err)
	}
	if err := os.WriteFile(archivePath, bodyData, 0o644); err != nil {
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

	body := `{"type":"script","files":["rna/accession.bigwig","dna/sample.cram"]}`
	created := createJob(t, app.server.URL+"/jobs", body)
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

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read script body: %v", err)
	}
	content := string(bodyData)
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

func TestLegacyScriptCreateEndpointIsRemoved(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	app := newTestApp(t, config)

	resp, err := http.Post(app.server.URL+"/script", "application/json", strings.NewReader(`{"files":["rna/accession.bigwig"]}`))
	if err != nil {
		t.Fatalf("post legacy script endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected legacy endpoint status %d, got %d: %s", http.StatusNotFound, resp.StatusCode, string(body))
	}
}

func TestCleanupRemovesExpiredJobAndArtifact(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.JobTTL = 120 * time.Millisecond
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "rna/accession.bigwig", "rna data")

	body := `{"type":"script","files":["rna/accession.bigwig"]}`
	created := createJob(t, app.server.URL+"/jobs", body)
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

func waitForDoneStatus(t *testing.T, baseURL, id string) api.JobStatusResponse {
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

		var job api.JobStatusResponse
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
	return api.JobStatusResponse{}
}
