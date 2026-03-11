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

func useTestOutputDir(t *testing.T) string {
	t.Helper()

	t.Cleanup(func() {
		core.LoadConfig()
	})

	outputDir := filepath.Join(t.TempDir(), "zips")
	t.Setenv("OUTPUT_DIR", outputDir)
	core.LoadConfig()

	if err := os.MkdirAll(core.OutputDir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}

	return outputDir
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
	useTestOutputDir(t)
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

	var got ZipResponse
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
		_ = os.Remove(filepath.Join(core.OutputDir, job.Filename))
	}
	store.Delete(got.ID)
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
	outputDir := useTestOutputDir(t)

	filename := "download-test.zip"
	zipPath := filepath.Join(outputDir, filename)
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
