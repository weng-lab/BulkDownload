package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jair/bulkdownload/api"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

type testApp struct {
	config appconfig.Config
	jobs   *jobs.Jobs
	server *httptest.Server
}

func newTestApp(t *testing.T, config appconfig.Config) testApp {
	t.Helper()

	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		t.Fatalf("create jobs dir: %v", err)
	}
	if config.SourceRootDir != "" {
		if err := os.MkdirAll(config.SourceRootDir, 0o755); err != nil {
			t.Fatalf("create source root dir: %v", err)
		}
	}

	jobStore := jobs.NewJobs()
	manager := service.NewManager(jobStore, config)
	stopCleanup := service.StartCleanup(jobStore, config.JobsDir, config.CleanupTick)
	t.Cleanup(stopCleanup)

	server := httptest.NewServer(api.NewRouter(slog.Default(), manager, jobStore, config))
	t.Cleanup(func() {
		server.Close()
	})

	return testApp{
		config: config,
		jobs:   jobStore,
		server: server,
	}
}

func testConfig(t *testing.T) appconfig.Config {
	t.Helper()

	root := t.TempDir()
	return appconfig.Config{
		JobsDir:         filepath.Join(root, "jobs"),
		SourceRootDir:   filepath.Join(root, "source"),
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		Port:            "0",
		AdminToken:      "test-admin-token",
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
	if job.Status != jobs.StatusDone || job.Progress != 100 {
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

	if job.Status != jobs.StatusDone || job.Progress != 100 {
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

func TestCreateJobRejectsInvalidTarballAndScriptRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		buildBody func(*testing.T, appconfig.Config) string
		wantBody  func(appconfig.Config) string
	}{
		{
			name: "tarball missing file",
			buildBody: func(_ *testing.T, _ appconfig.Config) string {
				return `{"type":"tarball","files":["nested/missing.txt"]}`
			},
			wantBody: func(_ appconfig.Config) string {
				return "file not found: nested/missing.txt\n"
			},
		},
		{
			name: "script absolute path",
			buildBody: func(t *testing.T, config appconfig.Config) string {
				path := writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
				return `{"type":"script","files":["` + path + `"]}`
			},
			wantBody: func(config appconfig.Config) string {
				return "absolute paths are not allowed: " + filepath.Join(config.SourceRootDir, "nested", "alpha.txt") + "\n"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testConfig(t)
			app := newTestApp(t, config)

			resp, err := http.Post(app.server.URL+"/jobs", "application/json", strings.NewReader(tt.buildBody(t, config)))
			if err != nil {
				t.Fatalf("create invalid job request: %v", err)
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
			if got, want := string(bodyData), tt.wantBody(config); got != want {
				t.Fatalf("expected body %q, got %q", want, got)
			}
		})
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
	if job.Status != jobs.StatusDone || job.Progress != 100 {
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
	if job.Status != jobs.StatusDone {
		t.Fatalf("expected completed script job, got %#v", job)
	}
	if job.Progress != 0 {
		t.Fatalf("expected script job progress to remain 0, got %#v", job)
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

func TestCleanupRemovesExpiredJobAndArtifactForSupportedJobTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		setupData func(*testing.T, appconfig.Config)
	}{
		{
			name: "zip",
			body: `{"type":"zip","files":["nested/alpha.txt","nested/bravo.txt"]}`,
			setupData: func(t *testing.T, config appconfig.Config) {
				t.Helper()
				writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
				writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")
			},
		},
		{
			name: "tarball",
			body: `{"type":"tarball","files":["nested/alpha.txt","nested/bravo.txt"]}`,
			setupData: func(t *testing.T, config appconfig.Config) {
				t.Helper()
				writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
				writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")
			},
		},
		{
			name: "script",
			body: `{"type":"script","files":["rna/accession.bigwig"]}`,
			setupData: func(t *testing.T, config appconfig.Config) {
				t.Helper()
				writeAppSourceFile(t, config.SourceRootDir, "rna/accession.bigwig", "rna data")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testConfig(t)
			config.JobTTL = 120 * time.Millisecond
			app := newTestApp(t, config)
			tt.setupData(t, config)

			created := createJob(t, app.server.URL+"/jobs", tt.body)
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
		})
	}
}

func TestAdminEndpointsExposeStoredMetadataAndHideExpiredJobs(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.JobTTL = 250 * time.Millisecond
	config.CleanupTick = time.Second
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")
	writeAppSourceFile(t, config.SourceRootDir, "rna/accession.bigwig", "rna data")

	zipCreated := createJob(t, app.server.URL+"/jobs", `{"type":"zip","files":["nested/alpha.txt","nested/bravo.txt"]}`)
	scriptCreated := createJob(t, app.server.URL+"/jobs", `{"type":"script","files":["rna/accession.bigwig"]}`)
	zipJob := waitForDoneStatus(t, app.server.URL, zipCreated.ID)
	scriptJob := waitForDoneStatus(t, app.server.URL, scriptCreated.ID)

	resp, err := adminGet(t, app, "/admin/jobs")
	if err != nil {
		t.Fatalf("get admin jobs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected admin jobs status %d, got %d: %s", http.StatusOK, resp.StatusCode, string(body))
	}

	var listed []api.AdminJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode admin list response: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 admin jobs, got %d", len(listed))
	}
	if listed[0].ID != scriptCreated.ID || listed[1].ID != zipCreated.ID {
		t.Fatalf("expected newest-first order, got %#v", listed)
	}
	assertAdminJobResponseShape(t, listed[0])
	assertAdminJobResponseShape(t, listed[1])
	if listed[0].InputSize <= 0 || listed[1].InputSize <= 0 {
		t.Fatalf("expected positive input sizes, got %#v", listed)
	}
	if listed[0].OutputSize != int64(len(downloadBody(t, app.server.URL+"/download/"+scriptJob.ID))) {
		t.Fatalf("script output size mismatch: %#v", listed[0])
	}
	if listed[1].OutputSize != int64(len(downloadBody(t, app.server.URL+"/download/"+zipJob.ID))) {
		t.Fatalf("zip output size mismatch: %#v", listed[1])
	}

	detailResp, err := adminGet(t, app, "/admin/jobs/"+zipCreated.ID)
	if err != nil {
		t.Fatalf("get admin job detail: %v", err)
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(detailResp.Body)
		t.Fatalf("expected admin job detail status %d, got %d: %s", http.StatusOK, detailResp.StatusCode, string(body))
	}
	var detail api.AdminJobResponse
	if err := json.NewDecoder(detailResp.Body).Decode(&detail); err != nil {
		t.Fatalf("decode admin detail response: %v", err)
	}
	assertAdminJobResponseShape(t, detail)
	if detail.ID != zipCreated.ID {
		t.Fatalf("expected detail for %q, got %#v", zipCreated.ID, detail)
	}

	time.Sleep(config.JobTTL + 50*time.Millisecond)

	expiredListResp, err := adminGet(t, app, "/admin/jobs")
	if err != nil {
		t.Fatalf("get expired admin jobs: %v", err)
	}
	defer expiredListResp.Body.Close()
	var expiredListed []api.AdminJobResponse
	if err := json.NewDecoder(expiredListResp.Body).Decode(&expiredListed); err != nil {
		t.Fatalf("decode expired admin list response: %v", err)
	}
	if len(expiredListed) != 0 {
		t.Fatalf("expected expired jobs to be hidden, got %#v", expiredListed)
	}

	expiredDetailResp, err := adminGet(t, app, "/admin/jobs/"+zipCreated.ID)
	if err != nil {
		t.Fatalf("get expired admin job detail: %v", err)
	}
	defer expiredDetailResp.Body.Close()
	if expiredDetailResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(expiredDetailResp.Body)
		t.Fatalf("expected expired admin job detail status %d, got %d: %s", http.StatusNotFound, expiredDetailResp.StatusCode, string(body))
	}
}

func TestAdminEndpointsRequireAdminToken(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(t))

	req, err := http.NewRequest(http.MethodGet, app.server.URL+"/admin/jobs", nil)
	if err != nil {
		t.Fatalf("build admin request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send admin request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected admin status %d, got %d: %s", http.StatusUnauthorized, resp.StatusCode, string(body))
	}
}

func TestAdminDeleteRemovesCompletedJobAndArtifact(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	config.JobTTL = 3 * time.Second
	config.CleanupTick = time.Second
	app := newTestApp(t, config)
	writeAppSourceFile(t, config.SourceRootDir, "nested/alpha.txt", "alpha contents")
	writeAppSourceFile(t, config.SourceRootDir, "nested/bravo.txt", "bravo contents")

	created := createJob(t, app.server.URL+"/jobs", `{"type":"zip","files":["nested/alpha.txt","nested/bravo.txt"]}`)
	job := waitForDoneStatus(t, app.server.URL, created.ID)
	artifactPath := filepath.Join(config.JobsDir, job.Filename)
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("expected artifact to exist before delete: %v", err)
	}

	req, err := http.NewRequest(http.MethodDelete, app.server.URL+"/admin/jobs/"+created.ID, nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}
	req.Header.Set("X-Admin-Token", app.config.AdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete admin job: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected delete status %d, got %d: %s", http.StatusNoContent, resp.StatusCode, string(body))
	}

	detailResp, err := adminGet(t, app, "/admin/jobs/"+created.ID)
	if err != nil {
		t.Fatalf("get deleted admin job detail: %v", err)
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(detailResp.Body)
		t.Fatalf("expected deleted admin job detail status %d, got %d: %s", http.StatusNotFound, detailResp.StatusCode, string(body))
	}

	listResp, err := adminGet(t, app, "/admin/jobs")
	if err != nil {
		t.Fatalf("get admin jobs after delete: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("expected admin jobs status %d, got %d: %s", http.StatusOK, listResp.StatusCode, string(body))
	}
	var listed []api.AdminJobResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode admin list response after delete: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected deleted job to be absent from admin list, got %#v", listed)
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact to be removed, stat err = %v", err)
	}
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

func adminGet(t *testing.T, app testApp, path string) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, app.server.URL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Admin-Token", app.config.AdminToken)

	return http.DefaultClient.Do(req)
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

		if job.Status == jobs.StatusDone {
			return job
		}
		if job.Status == jobs.StatusFailed {
			t.Fatalf("job %s failed: %s", id, job.Error)
		}

		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for job %s to complete", id)
	return api.JobStatusResponse{}
}

func downloadBody(t *testing.T, url string) []byte {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("download body request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected download status %d, got %d: %s", http.StatusOK, resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read download body: %v", err)
	}
	return body
}

func assertAdminJobResponseShape(t *testing.T, resp api.AdminJobResponse) {
	t.Helper()

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(raw) != 10 {
		t.Fatalf("field count = %d, want 10; raw = %#v", len(raw), raw)
	}
}
