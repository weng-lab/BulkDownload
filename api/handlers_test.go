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

	"github.com/go-chi/chi/v5"
	"github.com/jair/bulkdownload/core"
)

type handlerFixture struct {
	config  core.Config
	jobs    *core.Jobs
	manager *core.Manager
}

func newHandlerFixture(t *testing.T) handlerFixture {
	t.Helper()

	root := t.TempDir()
	config := core.Config{
		JobsDir:         filepath.Join(root, "jobs"),
		SourceRootDir:   filepath.Join(root, "source"),
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		Port:            "0",
		JobTTL:          3 * time.Second,
		CleanupTick:     time.Minute,
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("create jobs dir: %v", err)
	}
	if err := os.MkdirAll(config.SourceRootDir, 0o755); err != nil {
		t.Fatalf("create source root dir: %v", err)
	}

	jobs := core.NewJobs()
	return handlerFixture{
		config:  config,
		jobs:    jobs,
		manager: core.NewManager(jobs, config),
	}
}

func writeSourceFile(t *testing.T, root, relPath, contents string) string {
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

func TestHandleCreateJobAcceptsValidArchiveRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		jobType     string
		wantJobType core.JobType
	}{
		{
			name:        "zip",
			jobType:     "zip",
			wantJobType: core.JobTypeZip,
		},
		{
			name:        "tarball",
			jobType:     "tarball",
			wantJobType: core.JobTypeTarball,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newHandlerFixture(t)
			writeSourceFile(t, fixture.config.SourceRootDir, "nested/alpha.txt", "alpha contents")

			body := `{"type":"` + tt.jobType + `","files":["nested/alpha.txt"]}`
			req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
			rec := httptest.NewRecorder()

			HandleCreateJob(fixture.manager, fixture.config).ServeHTTP(rec, req)

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
			if got.ExpiresAt.IsZero() {
				t.Fatal("expected response to include expires_at")
			}

			job, ok := fixture.jobs.Get(got.ID)
			if !ok {
				t.Fatalf("expected job %q to be stored", got.ID)
			}
			if len(job.Files) != 1 || job.Files[0] != "nested/alpha.txt" {
				t.Fatalf("expected stored files [%q], got %#v", "nested/alpha.txt", job.Files)
			}
			if job.Type != tt.wantJobType {
				t.Fatalf("expected job type %q, got %q", tt.wantJobType, job.Type)
			}
			if time.Until(got.ExpiresAt) <= 0 {
				t.Fatal("expected expires_at to be in the future")
			}
		})
	}
}

func TestHandleCreateJobAcceptsValidScriptRequest(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	writeSourceFile(t, fixture.config.SourceRootDir, "rna/accession.bigwig", "rna data")
	writeSourceFile(t, fixture.config.SourceRootDir, "dna/sample.cram", "dna data")

	body := `{"type":"script","files":["rna/accession.bigwig","dna/sample.cram"]}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
	rec := httptest.NewRecorder()

	HandleCreateJob(fixture.manager, fixture.config).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	var got JobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	job, ok := fixture.jobs.Get(got.ID)
	if !ok {
		t.Fatalf("expected job %q to be stored", got.ID)
	}
	if want := []string{"rna/accession.bigwig", "dna/sample.cram"}; len(job.Files) != len(want) || job.Files[0] != want[0] || job.Files[1] != want[1] {
		t.Fatalf("expected normalized files %#v, got %#v", want, job.Files)
	}
	if job.Type != core.JobTypeScript {
		t.Fatalf("expected job type %q, got %q", core.JobTypeScript, job.Type)
	}

	waitForJobDone(t, fixture.jobs, got.ID)
}

func TestHandleCreateJobRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantCode    int
		wantBody    string
		prepareData func(*testing.T, handlerFixture)
	}{
		{
			name:     "invalid json",
			body:     `{`,
			wantCode: http.StatusBadRequest,
			wantBody: "invalid request body\n",
		},
		{
			name:     "missing type",
			body:     `{"files":["test.txt"]}`,
			wantCode: http.StatusBadRequest,
			wantBody: "type is required\n",
		},
		{
			name:     "empty files",
			body:     `{"type":"zip","files":[]}`,
			wantCode: http.StatusBadRequest,
			wantBody: "files list is empty\n",
		},
		{
			name:     "missing file",
			body:     `{"type":"zip","files":["nested/missing.txt"]}`,
			wantCode: http.StatusBadRequest,
			wantBody: "file not found: nested/missing.txt\n",
		},
		{
			name:     "absolute path",
			body:     `{"type":"zip","files":["/tmp/source/nested/alpha.txt"]}`,
			wantCode: http.StatusBadRequest,
			wantBody: "absolute paths are not allowed: /tmp/source/nested/alpha.txt\n",
		},
		{
			name:     "invalid job type",
			body:     `{"type":"invalid","files":["test.txt"]}`,
			wantCode: http.StatusBadRequest,
			wantBody: "invalid job type: invalid\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newHandlerFixture(t)
			if tt.prepareData != nil {
				tt.prepareData(t, fixture)
			}

			req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			HandleCreateJob(fixture.manager, fixture.config).ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("expected status %d, got %d", tt.wantCode, rec.Code)
			}
			if rec.Body.String() != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, rec.Body.String())
			}
		})
	}
}

func TestHandleStatusReturnsStoredJob(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	job := core.Job{
		ID:        "job-123",
		Type:      core.JobTypeZip,
		Status:    core.StatusProcessing,
		Progress:  37,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("add job: %v", err)
	}

	rec := performRouteRequest(http.MethodGet, "/status/{id}", "/status/"+job.ID, HandleStatus(fixture.jobs))

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

func TestHandleStatusReturnsNotFoundForUnknownJob(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	rec := performRouteRequest(http.MethodGet, "/status/{id}", "/status/missing", HandleStatus(fixture.jobs))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if rec.Body.String() != "job not found\n" {
		t.Fatalf("expected not found body, got %q", rec.Body.String())
	}
}

func TestHandleDownloadReturnsConflictUntilReady(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	job := core.Job{
		ID:        "job-pending",
		Type:      core.JobTypeZip,
		Status:    core.StatusProcessing,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("add job: %v", err)
	}

	rec := performRouteRequest(http.MethodGet, "/download/{id}", "/download/"+job.ID, HandleDownload(fixture.jobs, fixture.config))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestHandleDownloadServesFinishedArtifact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		body     string
	}{
		{name: "zip", filename: "download-test.zip", body: "zip bytes"},
		{name: "tarball", filename: "download-test.tar.gz", body: "tarball bytes"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newHandlerFixture(t)
			artifactPath := filepath.Join(fixture.config.JobsDir, tt.filename)
			if err := os.WriteFile(artifactPath, []byte(tt.body), 0o644); err != nil {
				t.Fatalf("write artifact: %v", err)
			}

			job := core.Job{
				ID:        "job-done",
				Type:      core.JobTypeZip,
				Status:    core.StatusDone,
				Filename:  tt.filename,
				ExpiresAt: time.Now().Add(time.Minute),
			}
			if err := fixture.jobs.Add(job); err != nil {
				t.Fatalf("add job: %v", err)
			}

			rec := performRouteRequest(http.MethodGet, "/download/{id}", "/download/"+job.ID, HandleDownload(fixture.jobs, fixture.config))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
			wantDisposition := `attachment; filename="` + tt.filename + `"`
			if got := rec.Header().Get("Content-Disposition"); got != wantDisposition {
				t.Fatalf("expected content disposition %q, got %q", wantDisposition, got)
			}
			if rec.Body.String() != tt.body {
				t.Fatalf("expected response body %q, got %q", tt.body, rec.Body.String())
			}
		})
	}
}

func waitForJobDone(t *testing.T, jobs *core.Jobs, id string) *core.Job {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := jobs.Get(id)
		if ok && job.Status == core.StatusDone && job.Filename != "" {
			return &job
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job %s to complete", id)
	return nil
}

func performRouteRequest(method, pattern, target string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.MethodFunc(method, pattern, handler)

	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
