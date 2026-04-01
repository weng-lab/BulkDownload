package service

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jair/bulkdownload/internal/artifacts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func (m *Manager) executeZipJob(ctx context.Context, jobID string) error {
	job, err := m.GetJobOfType(jobID, jobs.JobTypeZip)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".zip"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := artifacts.CreateZipFromRoot(ctx, outPath, m.sourceRootDir, job.Files, func(progress int) {
		_ = m.jobs.SetProgress(jobID, progress)
	}); err != nil {
		_ = artifacts.CleanupFile(outPath)
		wrappedErr := fmt.Errorf("create zip archive: %w", err)
		_ = m.jobs.MarkFailed(jobID, wrappedErr)
		return wrappedErr
	}

	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}

func (m *Manager) executeTarballJob(ctx context.Context, jobID string) error {
	job, err := m.GetJobOfType(jobID, jobs.JobTypeTarball)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".tar.gz"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := artifacts.CreateTarballFromRoot(ctx, outPath, m.sourceRootDir, job.Files, func(progress int) {
		_ = m.jobs.SetProgress(jobID, progress)
	}); err != nil {
		_ = artifacts.CleanupFile(outPath)
		wrappedErr := fmt.Errorf("create tarball archive: %w", err)
		_ = m.jobs.MarkFailed(jobID, wrappedErr)
		return wrappedErr
	}

	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}
