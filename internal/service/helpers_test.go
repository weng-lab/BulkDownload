package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
)

type testFixture struct {
	manager *Manager
	jobs    *jobs.Jobs
	config  appconfig.Config
}

func testConfig(t *testing.T) appconfig.Config {
	t.Helper()

	return appconfig.Config{
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
	jobStore := jobs.NewJobs()
	return testFixture{
		manager: NewManager(jobStore, config),
		jobs:    jobStore,
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
		"CLEANUP_TICK",
	} {
		key := key
		value, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%q) error = %v", key, err)
		}
		t.Cleanup(func() {
			var err error
			if ok {
				err = os.Setenv(key, value)
			} else {
				err = os.Unsetenv(key)
			}
			if err != nil {
				t.Fatalf("restore env %q: %v", key, err)
			}
		})
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
