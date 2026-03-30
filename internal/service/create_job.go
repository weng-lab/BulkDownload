package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jair/bulkdownload/internal/jobs"
)

var ErrCreateJobRequest = errors.New("create job request")

func (m *Manager) CreateJob(rawType string, requestedFiles []string) (jobs.Job, error) {
	jobType, err := parseCreateJobType(rawType)
	if err != nil {
		return jobs.Job{}, err
	}

	files, err := m.resolveCreateJobFiles(requestedFiles)
	if err != nil {
		return jobs.Job{}, err
	}

	job, err := m.createJob(jobType, files)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("create %s job: %w", jobType, err)
	}

	m.dispatchJob(job)

	return job, nil
}

func parseCreateJobType(raw string) (jobs.JobType, error) {
	jobType := jobs.JobType(raw)
	switch jobType {
	case jobs.JobTypeZip, jobs.JobTypeTarball, jobs.JobTypeScript:
		return jobType, nil
	default:
		return "", fmt.Errorf("%w: invalid job type: %s", ErrCreateJobRequest, raw)
	}
}

func (m *Manager) resolveCreateJobFiles(files []string) ([]string, error) {
	resolved := make([]string, 0, len(files))
	for _, rawPath := range files {
		file := strings.TrimSpace(rawPath)
		if file == "" {
			return nil, fmt.Errorf("%w: file path cannot be empty", ErrCreateJobRequest)
		}

		if filepath.IsAbs(file) {
			return nil, fmt.Errorf("%w: absolute paths are not allowed: %s", ErrCreateJobRequest, file)
		}

		checkPath := filepath.Join(m.sourceRootDir, file)
		if _, err := os.Stat(checkPath); err != nil {
			return nil, fmt.Errorf("%w: file not found: %s", ErrCreateJobRequest, file)
		}

		resolved = append(resolved, file)
	}

	return resolved, nil
}
