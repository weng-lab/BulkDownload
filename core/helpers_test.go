package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testFixture struct {
	manager *Manager
	jobs    *Jobs
	config  Config
}

func testConfig(t *testing.T) Config {
	t.Helper()

	return Config{
		JobsDir:         filepath.Join(t.TempDir(), "jobs"),
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		Port:            "0",
		JobTTL:          3 * time.Second,
		CleanupTick:     5 * time.Minute,
	}
}

func newTestFixture(t *testing.T) testFixture {
	t.Helper()

	config := testConfig(t)
	jobs := NewJobs()
	return testFixture{
		manager: NewManager(jobs, config),
		jobs:    jobs,
		config:  config,
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"JOBS_DIR",
		"SOURCE_ROOT_DIR",
		"PUBLIC_BASE_URL",
		"DOWNLOAD_ROOT_DIR",
		"PORT",
		"JOB_TTL",
		"ZIP_TTL",
		"CLEANUP_TICK",
	} {
		t.Setenv(key, "")
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working dir %q: %v", wd, err)
		}
	})
}
