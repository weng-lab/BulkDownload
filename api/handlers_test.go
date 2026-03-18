package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jair/bulkdownload/core"
)

func useTestJobsDir(t *testing.T) string {
	t.Helper()

	t.Cleanup(func() {
		core.LoadConfig()
	})

	jobsDir := filepath.Join(t.TempDir(), "jobs")
	t.Setenv("JOBS_DIR", jobsDir)
	t.Setenv("SOURCE_ROOT_DIR", "")
	t.Setenv("PUBLIC_BASE_URL", "https://download.mohd.org")
	t.Setenv("DOWNLOAD_ROOT_DIR", "mohd_data")
	core.LoadConfig()

	if err := os.MkdirAll(core.JobsDir, 0o755); err != nil {
		t.Fatalf("create jobs dir: %v", err)
	}

	return jobsDir
}

func useTestSourceRootDir(t *testing.T) string {
	t.Helper()

	t.Cleanup(func() {
		core.LoadConfig()
	})

	sourceRootDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(sourceRootDir, 0o755); err != nil {
		t.Fatalf("create source root dir: %v", err)
	}

	t.Setenv("SOURCE_ROOT_DIR", sourceRootDir)
	core.LoadConfig()

	return sourceRootDir
}

func TestHandleCreateZipReturnsExactMissingFileError(t *testing.T) {
	store := core.NewStore()
	req := httptest.NewRequest(http.MethodPost, "/zip", strings.NewReader(`{"files":["testdata/does-not-exist.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateZip(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	const wantBody = "file not found: testdata/does-not-exist.txt\n"
	if rec.Body.String() != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, rec.Body.String())
	}
}

func TestHandleCreateZipAcceptsValidRequest(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "alpha.txt")
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/zip", strings.NewReader(`{"files":["`+filePath+`"]}`))
	rec := httptest.NewRecorder()

	HandleCreateZip(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Fatalf("expected response to include job id")
	}
	if got.ExpiresAt.IsZero() {
		t.Fatalf("expected response to include expires_at")
	}

	job, ok := store.Get(got.ID)
	if !ok {
		t.Fatalf("expected job %q to be stored", got.ID)
	}
	if len(job.Files) != 1 || job.Files[0] != filePath {
		t.Fatalf("expected stored job files to contain %q, got %#v", filePath, job.Files)
	}
	if job.Status != core.StatusPending && job.Status != core.StatusProcessing {
		t.Fatalf("expected stored job status to be pending or processing, got %q", job.Status)
	}
	if time.Until(got.ExpiresAt) <= 0 {
		t.Fatalf("expected expires_at to be in the future")
	}
	if job.Filename != "" {
		t.Fatalf("expected filename to be empty before zip completes, got %q", job.Filename)
	}
	if job.Error != "" {
		t.Fatalf("expected error to be empty, got %q", job.Error)
	}
	if job.Filename != "" {
		_ = os.Remove(filepath.Join(core.JobsDir, job.Filename))
	}
	store.Delete(got.ID)
}

func TestHandleCreateZipResolvesRelativePathFromSourceRoot(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)
	sourceRootDir := useTestSourceRootDir(t)

	filePath := filepath.Join(sourceRootDir, "nested", "alpha.txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create nested source dir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/zip", strings.NewReader(`{"files":["nested/alpha.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateZip(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	job, ok := store.Get(got.ID)
	if !ok {
		t.Fatalf("expected job %q to be stored", got.ID)
	}
	if len(job.Files) != 1 || job.Files[0] != filePath {
		t.Fatalf("expected stored job files to contain %q, got %#v", filePath, job.Files)
	}

	store.Delete(got.ID)
}

func TestHandleCreateZipRejectsPathOutsideSourceRoot(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)
	useTestSourceRootDir(t)

	req := httptest.NewRequest(http.MethodPost, "/zip", strings.NewReader(`{"files":["../secret.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateZip(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	const wantBody = "file path cannot escape source root: ../secret.txt\n"
	if rec.Body.String() != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, rec.Body.String())
	}
}

func TestHandleCreateTarballReturnsExactMissingFileError(t *testing.T) {
	store := core.NewStore()
	req := httptest.NewRequest(http.MethodPost, "/tarball", strings.NewReader(`{"files":["testdata/does-not-exist.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateTarball(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	const wantBody = "file not found: testdata/does-not-exist.txt\n"
	if rec.Body.String() != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, rec.Body.String())
	}
}

func TestHandleCreateTarballAcceptsValidRequest(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "alpha.txt")
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tarball", strings.NewReader(`{"files":["`+filePath+`"]}`))
	rec := httptest.NewRecorder()

	HandleCreateTarball(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Fatalf("expected response to include job id")
	}
	if got.ExpiresAt.IsZero() {
		t.Fatalf("expected response to include expires_at")
	}

	job, ok := store.Get(got.ID)
	if !ok {
		t.Fatalf("expected job %q to be stored", got.ID)
	}
	if len(job.Files) != 1 || job.Files[0] != filePath {
		t.Fatalf("expected stored job files to contain %q, got %#v", filePath, job.Files)
	}
	if job.Status != core.StatusPending && job.Status != core.StatusProcessing {
		t.Fatalf("expected stored job status to be pending or processing, got %q", job.Status)
	}
	if time.Until(got.ExpiresAt) <= 0 {
		t.Fatalf("expected expires_at to be in the future")
	}
	if job.Filename != "" {
		t.Fatalf("expected filename to be empty before tarball completes, got %q", job.Filename)
	}
	if job.Error != "" {
		t.Fatalf("expected error to be empty, got %q", job.Error)
	}
	if job.Filename != "" {
		_ = os.Remove(filepath.Join(core.JobsDir, job.Filename))
	}
	store.Delete(got.ID)
}

func TestHandleCreateTarballResolvesRelativePathFromSourceRoot(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)
	sourceRootDir := useTestSourceRootDir(t)

	filePath := filepath.Join(sourceRootDir, "nested", "alpha.txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create nested source dir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("alpha contents"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tarball", strings.NewReader(`{"files":["nested/alpha.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateTarball(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	job, ok := store.Get(got.ID)
	if !ok {
		t.Fatalf("expected job %q to be stored", got.ID)
	}
	if len(job.Files) != 1 || job.Files[0] != filePath {
		t.Fatalf("expected stored job files to contain %q, got %#v", filePath, job.Files)
	}

	store.Delete(got.ID)
}

func TestHandleCreateTarballRejectsPathOutsideSourceRoot(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)
	useTestSourceRootDir(t)

	req := httptest.NewRequest(http.MethodPost, "/tarball", strings.NewReader(`{"files":["../secret.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateTarball(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	const wantBody = "file path cannot escape source root: ../secret.txt\n"
	if rec.Body.String() != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, rec.Body.String())
	}
}

func TestHandleStatusReturnsStoredJob(t *testing.T) {
	store := core.NewStore()
	job := &core.Job{ID: "job-123", Status: core.StatusProcessing, ExpiresAt: time.Now().Add(time.Minute)}
	store.Set(job)

	req := httptest.NewRequest(http.MethodGet, "/status/"+job.ID, nil)
	rec := httptest.NewRecorder()

	HandleStatus(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got core.Job
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != job.ID {
		t.Fatalf("expected job id %q, got %q", job.ID, got.ID)
	}
	if got.Status != job.Status {
		t.Fatalf("expected job status %q, got %q", job.Status, got.Status)
	}
}

func TestHandleDownloadServesFinishedZip(t *testing.T) {
	store := core.NewStore()
	jobsDir := useTestJobsDir(t)

	filename := "download-test.zip"
	zipPath := filepath.Join(jobsDir, filename)
	t.Cleanup(func() {
		_ = os.Remove(zipPath)
	})

	if err := os.WriteFile(zipPath, []byte("zip bytes"), 0o644); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	job := &core.Job{
		ID:        "job-done",
		Status:    core.StatusDone,
		Filename:  filename,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	store.Set(job)

	req := httptest.NewRequest(http.MethodGet, "/download/"+job.ID, nil)
	rec := httptest.NewRecorder()

	HandleDownload(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	wantDisposition := `attachment; filename="` + filename + `"`
	if got := rec.Header().Get("Content-Disposition"); got != wantDisposition {
		t.Fatalf("expected content disposition %q, got %q", wantDisposition, got)
	}
	if rec.Body.String() != "zip bytes" {
		t.Fatalf("expected response body %q, got %q", "zip bytes", rec.Body.String())
	}
}

func TestHandleDownloadServesFinishedTarball(t *testing.T) {
	store := core.NewStore()
	jobsDir := useTestJobsDir(t)

	filename := "download-test.tar.gz"
	tarballPath := filepath.Join(jobsDir, filename)
	t.Cleanup(func() {
		_ = os.Remove(tarballPath)
	})

	if err := os.WriteFile(tarballPath, []byte("tarball bytes"), 0o644); err != nil {
		t.Fatalf("write tarball file: %v", err)
	}

	job := &core.Job{
		ID:        "job-tarball",
		Status:    core.StatusDone,
		Filename:  filename,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	store.Set(job)

	req := httptest.NewRequest(http.MethodGet, "/download/"+job.ID, nil)
	rec := httptest.NewRecorder()

	HandleDownload(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	wantDisposition := `attachment; filename="` + filename + `"`
	if got := rec.Header().Get("Content-Disposition"); got != wantDisposition {
		t.Fatalf("expected content disposition %q, got %q", wantDisposition, got)
	}
	if rec.Body.String() != "tarball bytes" {
		t.Fatalf("expected response body %q, got %q", "tarball bytes", rec.Body.String())
	}
}

func TestHandleCreateScriptAcceptsValidRequest(t *testing.T) {
	store := core.NewStore()
	jobsDir := useTestJobsDir(t)

	req := httptest.NewRequest(http.MethodPost, "/script", strings.NewReader(`{"files":["rna/accession.bigwig","rna/second.bigwig"]}`))
	rec := httptest.NewRecorder()

	HandleCreateScript(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	job, ok := store.Get(got.ID)
	if !ok {
		t.Fatalf("expected job %q to be stored", got.ID)
	}
	if len(job.Files) != 2 || job.Files[0] != "rna/accession.bigwig" || job.Files[1] != "rna/second.bigwig" {
		t.Fatalf("expected normalized relative paths to be stored, got %#v", job.Files)
	}

	waitForScriptJobDone(t, store, got.ID)

	job, _ = store.Get(got.ID)
	if job.Filename == "" {
		t.Fatalf("expected script filename to be set")
	}
	if _, err := os.Stat(filepath.Join(jobsDir, job.Filename)); err != nil {
		t.Fatalf("expected generated script to exist: %v", err)
	}
	if time.Until(got.ExpiresAt) <= 0 {
		t.Fatalf("expected expires_at to be in the future")
	}

	store.Delete(got.ID)
}

func TestHandleCreateScriptRejectsUnsafePath(t *testing.T) {
	store := core.NewStore()
	useTestJobsDir(t)

	req := httptest.NewRequest(http.MethodPost, "/script", strings.NewReader(`{"files":["../secret.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateScript(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	const wantBody = "file path cannot escape the download root: ../secret.txt\n"
	if rec.Body.String() != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, rec.Body.String())
	}
}

func waitForScriptJobDone(t *testing.T, store *core.Store, id string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := store.Get(id)
		if ok && job.Status == core.StatusDone && job.Filename != "" {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for script job %s to complete", id)
}
