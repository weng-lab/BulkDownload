package core

import (
	"path/filepath"
	"testing"
	"time"
)

func testConfig(t *testing.T) Config {
	t.Helper()

	return Config{
		JobsDir:         filepath.Join(t.TempDir(), "jobs"),
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		Port:            "8080",
		JobTTL:          3 * time.Second,
		CleanupTick:     5 * time.Minute,
	}
}

func testManager(t *testing.T) (*Manager, *Jobs, Config) {
	t.Helper()

	config := testConfig(t)
	jobs := NewJobs()
	manager := NewManager(jobs, config)
	return manager, jobs, config
}
