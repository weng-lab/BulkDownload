package service

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jair/bulkdownload/internal/artifacts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func (m *Manager) executeScriptJob(ctx context.Context, jobID string) error {
	job, err := m.GetJobOfType(jobID, jobs.JobTypeScript)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".sh"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := artifacts.CreateDownloadScript(ctx, outPath, m.publicBaseURL, m.downloadRootDir, job.Files); err != nil {
		_ = artifacts.CleanupFile(outPath)
		wrappedErr := fmt.Errorf("create download script: %w", err)
		_ = m.jobs.MarkFailed(jobID, wrappedErr)
		return wrappedErr
	}

	outputSize, err := outputFileSize(outPath)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkDone(jobID, filename, outputSize); err != nil {
		return err
	}

	return nil
}
