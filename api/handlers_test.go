package api

import (
	"encoding/json"
	"errors"
	"log/slog"
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
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

type handlerFixture struct {
	config  appconfig.Config
	jobs    *jobs.Jobs
	manager *service.Manager
}

func newHandlerFixture(t *testing.T) handlerFixture {
	t.Helper()

	root := t.TempDir()
	config := appconfig.Config{
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

	jobStore := jobs.NewJobs()
	return handlerFixture{
		config:  config,
		jobs:    jobStore,
		manager: service.NewManager(jobStore, config),
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

func TestHandleCreateJobReturnsAcceptedResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		jobType     string
		files       []string
		prepareData func(*testing.T, handlerFixture)
	}{
		{
			name:    "zip",
			jobType: "zip",
			files:   []string{"nested/alpha.txt"},
			prepareData: func(t *testing.T, fixture handlerFixture) {
				t.Helper()
				writeSourceFile(t, fixture.config.SourceRootDir, "nested/alpha.txt", "alpha contents")
			},
		},
		{
			name:    "tarball",
			jobType: "tarball",
			files:   []string{"nested/alpha.txt", "nested/bravo.txt"},
			prepareData: func(t *testing.T, fixture handlerFixture) {
				t.Helper()
				writeSourceFile(t, fixture.config.SourceRootDir, "nested/alpha.txt", "alpha contents")
				writeSourceFile(t, fixture.config.SourceRootDir, "nested/bravo.txt", "bravo contents")
			},
		},
		{
			name:    "script",
			jobType: "script",
			files:   []string{"rna/accession.bigwig", "dna/sample.cram"},
			prepareData: func(t *testing.T, fixture handlerFixture) {
				t.Helper()
				writeSourceFile(t, fixture.config.SourceRootDir, "rna/accession.bigwig", "rna data")
				writeSourceFile(t, fixture.config.SourceRootDir, "dna/sample.cram", "dna data")
			},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newHandlerFixture(t)
			if tc.prepareData != nil {
				tc.prepareData(t, fixture)
			}

			body := `{"type":"` + tc.jobType + `","files":["` + strings.Join(tc.files, `","`) + `"]}`
			req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(body))
			rec := httptest.NewRecorder()

			HandleCreateJob(fixture.manager, fixture.config).ServeHTTP(rec, req)

			if rec.Code != http.StatusAccepted {
				t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Fatalf("expected JSON content type, got %q", got)
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
			if time.Until(got.ExpiresAt) <= 0 {
				t.Fatal("expected expires_at to be in the future")
			}
		})
	}
}

func TestHandleCreateJobRejectsTransportErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		wantCode int
		wantBody string
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
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newHandlerFixture(t)

			req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			HandleCreateJob(fixture.manager, fixture.config).ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("expected status %d, got %d", tc.wantCode, rec.Code)
			}
			if rec.Body.String() != tc.wantBody {
				t.Fatalf("expected body %q, got %q", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestHandleCreateJobMapsServiceValidationErrorsToBadRequest(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"type":"invalid","files":["nested/alpha.txt"]}`))
	rec := httptest.NewRecorder()

	HandleCreateJob(fixture.manager, fixture.config).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if rec.Body.String() != "invalid job type: invalid\n" {
		t.Fatalf("expected invalid type error body, got %q", rec.Body.String())
	}
}

func TestWriteCreateJobErrorMapsServiceFailures(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	writeSourceFile(t, fixture.config.SourceRootDir, "nested/alpha.txt", "alpha contents")

	_, validationErr := fixture.manager.CreateJob("invalid", []string{"nested/alpha.txt"})
	if validationErr == nil {
		t.Fatal("CreateJob() error = nil, want non-nil")
	}

	tests := []struct {
		name          string
		requestedType string
		err           error
		wantCode      int
		wantBody      string
	}{
		{
			name:          "request validation error",
			requestedType: "invalid",
			err:           validationErr,
			wantCode:      http.StatusBadRequest,
			wantBody:      "invalid job type: invalid\n",
		},
		{
			name:          "unexpected service failure",
			requestedType: "zip",
			err:           errors.New("boom"),
			wantCode:      http.StatusInternalServerError,
			wantBody:      "failed to dispatch job\n",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			writeCreateJobError(rec, slog.Default(), tc.requestedType, tc.err)

			if rec.Code != tc.wantCode {
				t.Fatalf("expected status %d, got %d", tc.wantCode, rec.Code)
			}
			if rec.Body.String() != tc.wantBody {
				t.Fatalf("expected body %q, got %q", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestHandleStatusReturnsStoredJob(t *testing.T) {
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
	job := jobs.Job{
		ID:        "job-pending",
		Type:      jobs.JobTypeZip,
		Status:    jobs.StatusProcessing,
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
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newHandlerFixture(t)
			artifactPath := filepath.Join(fixture.config.JobsDir, tc.filename)
			if err := os.WriteFile(artifactPath, []byte(tc.body), 0o644); err != nil {
				t.Fatalf("write artifact: %v", err)
			}

			job := jobs.Job{
				ID:        "job-done",
				Type:      jobs.JobTypeZip,
				Status:    jobs.StatusDone,
				Filename:  tc.filename,
				ExpiresAt: time.Now().Add(time.Minute),
			}
			if err := fixture.jobs.Add(job); err != nil {
				t.Fatalf("add job: %v", err)
			}

			rec := performRouteRequest(http.MethodGet, "/download/{id}", "/download/"+job.ID, HandleDownload(fixture.jobs, fixture.config))

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
			wantDisposition := `attachment; filename="` + tc.filename + `"`
			if got := rec.Header().Get("Content-Disposition"); got != wantDisposition {
				t.Fatalf("expected content disposition %q, got %q", wantDisposition, got)
			}
			if rec.Body.String() != tc.body {
				t.Fatalf("expected response body %q, got %q", tc.body, rec.Body.String())
			}
		})
	}
}

func TestHandleAdminListJobsReturnsVisibleJobsNewestFirst(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	now := time.Now()
	jobsToAdd := []jobs.Job{
		{
			ID:         "older",
			Type:       jobs.JobTypeZip,
			Status:     jobs.StatusDone,
			Progress:   100,
			Files:      []string{"nested/alpha.txt"},
			InputSize:  12,
			OutputSize: 34,
			CreatedAt:  now.Add(-2 * time.Minute),
			ExpiresAt:  now.Add(time.Minute),
		},
		{
			ID:         "newer",
			Type:       jobs.JobTypeScript,
			Status:     jobs.StatusFailed,
			Progress:   0,
			Files:      []string{"rna/accession.bigwig"},
			InputSize:  56,
			OutputSize: 0,
			CreatedAt:  now.Add(-time.Minute),
			ExpiresAt:  now.Add(time.Minute),
			Error:      "boom",
		},
		{
			ID:        "expired",
			Type:      jobs.JobTypeTarball,
			Status:    jobs.StatusDone,
			CreatedAt: now.Add(-3 * time.Minute),
			ExpiresAt: now.Add(-time.Second),
		},
	}
	for _, job := range jobsToAdd {
		if err := fixture.jobs.Add(job); err != nil {
			t.Fatalf("add job %q: %v", job.ID, err)
		}
	}

	rec := performRouteRequest(http.MethodGet, "/admin/jobs", "/admin/jobs", HandleAdminListJobs(fixture.jobs))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got []AdminJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 visible jobs, got %d", len(got))
	}
	if got[0].ID != "newer" || got[1].ID != "older" {
		t.Fatalf("unexpected order: %#v", got)
	}
	if diff := compareSortedFiles([]string{"rna/accession.bigwig"}, got[0].Files); diff != "" {
		t.Errorf("files mismatch (-want +got):\n%s", diff)
	}
	assertAdminJobFieldCount(t, got[0], 10)
}

func TestHandleAdminGetJobReturnsStoredJob(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	now := time.Now()
	job := jobs.Job{
		ID:         "job-123",
		Type:       jobs.JobTypeZip,
		Status:     jobs.StatusProcessing,
		Progress:   37,
		Files:      []string{"nested/alpha.txt", "nested/bravo.txt"},
		InputSize:  123,
		OutputSize: 0,
		CreatedAt:  now.Add(-time.Minute),
		ExpiresAt:  now.Add(time.Minute),
		Error:      "",
	}
	if err := fixture.jobs.Add(job); err != nil {
		t.Fatalf("add job: %v", err)
	}

	rec := performRouteRequest(http.MethodGet, "/admin/jobs/{id}", "/admin/jobs/"+job.ID, HandleAdminGetJob(fixture.jobs))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got AdminJobResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != job.ID || got.Type != job.Type || got.Status != job.Status || got.Progress != job.Progress {
		t.Fatalf("unexpected job response: %#v", got)
	}
	if diff := compareSortedFiles(job.Files, got.Files); diff != "" {
		t.Errorf("files mismatch (-want +got):\n%s", diff)
	}
	if got.InputSize != job.InputSize || got.OutputSize != job.OutputSize {
		t.Fatalf("unexpected size metadata: %#v", got)
	}
	assertAdminJobFieldCount(t, got, 10)
}

func TestHandleAdminGetJobHidesExpiredAndMissingJobs(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t)
	expired := jobs.Job{
		ID:        "expired",
		Type:      jobs.JobTypeZip,
		CreatedAt: time.Now().Add(-time.Minute),
		ExpiresAt: time.Now().Add(-time.Second),
	}
	if err := fixture.jobs.Add(expired); err != nil {
		t.Fatalf("add expired job: %v", err)
	}

	for _, target := range []string{"/admin/jobs/missing", "/admin/jobs/expired"} {
		rec := performRouteRequest(http.MethodGet, "/admin/jobs/{id}", target, HandleAdminGetJob(fixture.jobs))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("target %q expected status %d, got %d", target, http.StatusNotFound, rec.Code)
		}
		if rec.Body.String() != "job not found\n" {
			t.Fatalf("target %q expected not found body, got %q", target, rec.Body.String())
		}
	}
}

func compareSortedFiles(want, got []string) string {
	want = append([]string(nil), want...)
	got = append([]string(nil), got...)
	sort.Strings(want)
	sort.Strings(got)
	return cmp.Diff(want, got)
}

func assertAdminJobFieldCount(t *testing.T, resp AdminJobResponse, want int) {
	t.Helper()

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(raw) != want {
		t.Fatalf("field count = %d, want %d; raw = %#v", len(raw), want, raw)
	}
}

func performRouteRequest(method, pattern, target string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.MethodFunc(method, pattern, handler)

	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
