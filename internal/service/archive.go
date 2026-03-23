package service

import (
	"path/filepath"

	"github.com/jair/bulkdownload/internal/artifacts"
	"github.com/jair/bulkdownload/internal/jobs"
)

func (m *Manager) executeZipJob(jobID string) error {
	job, err := m.GetJobOfType(jobID, jobs.JobTypeZip)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".zip"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := artifacts.CreateZipFromRoot(outPath, m.sourceRootDir, job.Files, func(progress int) {
		_ = m.jobs.SetProgress(jobID, progress)
	}); err != nil {
		_ = artifacts.CleanupFile(outPath)
		_ = m.jobs.MarkFailed(jobID, err)
		return err
	}

	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}

func (m *Manager) executeTarballJob(jobID string) error {
	job, err := m.GetJobOfType(jobID, jobs.JobTypeTarball)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".tar.gz"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := artifacts.CreateTarballFromRoot(outPath, m.sourceRootDir, job.Files, func(progress int) {
		_ = m.jobs.SetProgress(jobID, progress)
	}); err != nil {
		_ = artifacts.CleanupFile(outPath)
		_ = m.jobs.MarkFailed(jobID, err)
		return err
	}

	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}
