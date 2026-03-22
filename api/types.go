package api

import (
	"time"

	"github.com/jair/bulkdownload/core"
)

type JobRequest struct {
	Files []string `json:"files"`
}

type JobResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type JobStatusResponse struct {
	ID        string         `json:"id"`
	Type      core.JobType   `json:"type"`
	Status    core.JobStatus `json:"status"`
	Progress  int            `json:"progress"`
	ExpiresAt time.Time      `json:"expires_at"`
	Error     string         `json:"error,omitempty"`
	Filename  string         `json:"filename,omitempty"`
}

func newJobStatusResponse(job core.Job) JobStatusResponse {
	return JobStatusResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    job.Status,
		Progress:  job.Progress,
		ExpiresAt: job.ExpiresAt,
		Error:     job.Error,
		Filename:  job.Filename,
	}
}
