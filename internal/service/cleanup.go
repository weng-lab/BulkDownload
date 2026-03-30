package service

import (
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/jair/bulkdownload/internal/artifacts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func sweepExpired(jobStore *jobs.Jobs, jobsDir string, now time.Time) {
	logger := slog.Default()
	expiredJobs := jobStore.Expired(now)
	removed := 0

	for _, job := range expiredJobs {
		jobLogger := logger.With("job_id", job.ID, "job_type", job.Type)

		if job.Filename != "" {
			filePath := filepath.Join(jobsDir, job.Filename)
			if err := artifacts.CleanupFile(filePath); err != nil {
				jobLogger.Error("cleanup file removal failed", "filename", job.Filename, "error", err)
			} else {
				jobLogger.Info("cleanup removed expired job", "filename", job.Filename)
			}
		} else {
			jobLogger.Info("cleanup removed expired job")
		}

		jobStore.Delete(job.ID)
		removed++
	}

	if removed > 0 {
		logger.Info("cleanup sweep removed expired jobs", "removed_count", removed)
	}
}

func StartCleanup(jobStore *jobs.Jobs, jobsDir string, interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	stopCh := make(chan struct{})
	done := make(chan struct{})
	var once sync.Once

	go func() {
		defer ticker.Stop()
		defer close(done)
		for {
			select {
			case <-stopCh:
				return
			case now := <-ticker.C:
				slog.Default().Debug("cleanup sweep started", "jobs_dir", jobsDir)
				sweepExpired(jobStore, jobsDir, now)
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(stopCh)
			<-done
		})
	}
}
