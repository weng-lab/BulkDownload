package core

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/jair/bulkdownload/internal/jobs"
)

func sweepExpired(jobStore *jobs.Jobs, jobsDir string, now time.Time) {
	for _, job := range jobStore.Expired(now) {
		if job.Filename != "" {
			_ = cleanupFile(filepath.Join(jobsDir, job.Filename))
		}
		jobStore.Delete(job.ID)
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
