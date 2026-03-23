package core

import (
	"path/filepath"

	"github.com/jair/bulkdownload/internal/artifacts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func (m *Manager) executeScriptJob(jobID string) error {
	job, err := m.GetJobOfType(jobID, jobs.JobTypeScript)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".sh"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := artifacts.CreateDownloadScript(outPath, m.publicBaseURL, m.downloadRootDir, job.Files); err != nil {
		_ = artifacts.CleanupFile(outPath)
		_ = m.jobs.MarkFailed(jobID, err)
		return err
	}
	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}
