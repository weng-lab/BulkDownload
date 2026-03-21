package core

import (
	"context"
	"path/filepath"
	"time"
)

func SweepExpired(jobs *Jobs, jobsDir string, now time.Time) {
	for _, job := range jobs.Expired(now) {
		if job.Filename != "" {
			_ = cleanupFile(filepath.Join(jobsDir, job.Filename))
		}
		jobs.Delete(job.ID)
	}
}

func StartCleanup(ctx context.Context, jobs *Jobs, jobsDir string, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				SweepExpired(jobs, jobsDir, now)
			}
		}
	}()
}
