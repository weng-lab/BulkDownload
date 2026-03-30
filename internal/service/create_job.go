package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jair/bulkdownload/internal/jobs"
)

type CreateJobRequestError struct {
	message string
}

func (e *CreateJobRequestError) Error() string {
	return e.message
}

func IsCreateJobRequestError(err error) bool {
	var target *CreateJobRequestError
	return errors.As(err, &target)
}

func (m *Manager) CreateJob(rawType string, requestedFiles []string) (jobs.Job, error) {
	jobType, err := parseCreateJobType(rawType)
	if err != nil {
		return jobs.Job{}, err
	}

	files, err := m.resolveCreateJobFiles(requestedFiles)
	if err != nil {
		return jobs.Job{}, err
	}

	return m.createAndDispatchJob(jobType, files)
}

func parseCreateJobType(raw string) (jobs.JobType, error) {
	jobType := jobs.JobType(raw)
	switch jobType {
	case jobs.JobTypeZip, jobs.JobTypeTarball, jobs.JobTypeScript:
		return jobType, nil
	default:
		return "", newCreateJobRequestError("invalid job type: %s", raw)
	}
}

func (m *Manager) resolveCreateJobFiles(files []string) ([]string, error) {
	resolved := make([]string, 0, len(files))
	for _, rawPath := range files {
		file := strings.TrimSpace(rawPath)
		if file == "" {
			return nil, newCreateJobRequestError("file path cannot be empty")
		}

		if filepath.IsAbs(file) {
			return nil, newCreateJobRequestError("absolute paths are not allowed: %s", file)
		}

		checkPath := filepath.Join(m.sourceRootDir, file)
		if _, err := os.Stat(checkPath); err != nil {
			return nil, newCreateJobRequestError("file not found: %s", file)
		}

		resolved = append(resolved, file)
	}

	return resolved, nil
}

func newCreateJobRequestError(format string, args ...any) error {
	return &CreateJobRequestError{message: fmt.Sprintf(format, args...)}
}
