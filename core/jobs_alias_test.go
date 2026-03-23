package core

import jobstore "github.com/jair/bulkdownload/internal/jobs"

type Job = jobstore.Job
type Jobs = jobstore.Jobs
type JobType = jobstore.JobType
type JobStatus = jobstore.JobStatus

const (
	StatusPending    = jobstore.StatusPending
	StatusProcessing = jobstore.StatusProcessing
	StatusDone       = jobstore.StatusDone
	StatusFailed     = jobstore.StatusFailed
	JobTypeZip       = jobstore.JobTypeZip
	JobTypeTarball   = jobstore.JobTypeTarball
	JobTypeScript    = jobstore.JobTypeScript
)

var (
	ErrJobExists    = jobstore.ErrJobExists
	ErrJobNotFound  = jobstore.ErrJobNotFound
	ErrInvalidJobID = jobstore.ErrInvalidJobID
)

func NewJobs() *Jobs {
	return jobstore.NewJobs()
}
