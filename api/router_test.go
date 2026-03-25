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

	"github.com/jair/bulkdownload/internal/jobs"
)

func TestNewRouterWiresCreateJobRoute(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	writeSourceFile(t, fixture.config.SourceRootDir, "nested/alpha.txt", "alpha contents")

	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"type":"zip","files":["nested/alpha.txt"]}`))
	rec := httptest.NewRecorder()

	NewRouter(fixture.manager, fixture.jobs, fixture.config).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Fatal("expected response to include job id")
	}
}

func TestNewRouterWiresStatusRoute(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	job := jobs.Job{
		ID:        "job-123",
		Type:      jobs.JobTypeZip,
		Status:    jobs.StatusProcessing,
		Progress:  37,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("add job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/status/"+job.ID, nil)
	rec := httptest.NewRecorder()

	NewRouter(fixture.manager, fixture.jobs, fixture.config).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got JobStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != job.ID || got.Type != job.Type || got.Status != job.Status || got.Progress != job.Progress {
		t.Fatalf("unexpected job response: %#v", got)
	}
}

func TestNewRouterWiresDownloadRoute(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	filename := "download-test.zip"
	artifactPath := filepath.Join(fixture.config.JobsDir, filename)
	if err := os.WriteFile(artifactPath, []byte("zip bytes"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	job := jobs.Job{
		ID:        "job-done",
		Type:      jobs.JobTypeZip,
		Status:    jobs.StatusDone,
		Filename:  filename,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("add job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/download/"+job.ID, nil)
	rec := httptest.NewRecorder()

	NewRouter(fixture.manager, fixture.jobs, fixture.config).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="`+filename+`"` {
		t.Fatalf("unexpected content disposition %q", got)
	}
	if rec.Body.String() != "zip bytes" {
		t.Fatalf("expected response body %q, got %q", "zip bytes", rec.Body.String())
	}
}

func TestNewRouterAppliesCORSMiddleware(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	req := httptest.NewRequest(http.MethodOptions, "/jobs", nil)
	req.Header.Set("Origin", "https://client.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()

	NewRouter(fixture.manager, fixture.jobs, fixture.config).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected allow origin %q, got %q", "*", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Fatalf("expected allow methods to include %q, got %q", http.MethodPost, got)
	}
}
