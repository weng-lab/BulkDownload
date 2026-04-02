package api

import (
	"time"

	"github.com/jair/bulkdownload/internal/jobs"
)

type CreateJobRequest struct {
	Type  string   `json:"type"`
	Files []string `json:"files"`
}

type JobResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type JobStatusResponse struct {
	ID        string         `json:"id"`
	Type      jobs.JobType   `json:"type"`
	Status    jobs.JobStatus `json:"status"`
	Progress  int            `json:"progress"`
	ExpiresAt time.Time      `json:"expires_at"`
	Error     string         `json:"error,omitempty"`
	Filename  string         `json:"filename,omitempty"`
}

type AdminJobResponse struct {
	ID           string         `json:"id"`
	Type         jobs.JobType   `json:"type"`
	Status       jobs.JobStatus `json:"status"`
	Progress     int            `json:"progress"`
	Files        []string       `json:"files"`
	InputSize    int64          `json:"input_size"`
	OutputSize   int64          `json:"output_size"`
	CreationTime time.Time      `json:"creation_time"`
	ExpiresAt    time.Time      `json:"expires_at"`
	Error        string         `json:"error"`
}

func newJobStatusResponse(job jobs.Job) JobStatusResponse {
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

func newAdminJobResponse(job jobs.Job) AdminJobResponse {
	return AdminJobResponse{
		ID:           job.ID,
		Type:         job.Type,
		Status:       job.Status,
		Progress:     job.Progress,
		Files:        append([]string(nil), job.Files...),
		InputSize:    job.InputSize,
		OutputSize:   job.OutputSize,
		CreationTime: job.CreationTime,
		ExpiresAt:    job.ExpiresAt,
		Error:        job.Error,
	}
}
