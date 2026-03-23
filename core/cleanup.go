package core

import (
	"path/filepath"
	"sync"
	"time"
)

func sweepExpired(jobs *Jobs, jobsDir string, now time.Time) {
	for _, job := range jobs.Expired(now) {
		if job.Filename != "" {
			_ = cleanupFile(filepath.Join(jobsDir, job.Filename))
		}
		jobs.Delete(job.ID)
	}
}

func StartCleanup(jobs *Jobs, jobsDir string, interval time.Duration) func() {
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
				sweepExpired(jobs, jobsDir, now)
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
