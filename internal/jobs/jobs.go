package jobs

import (
	"errors"
	"slices"
	"sync"
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusDone       JobStatus = "done"
	StatusFailed     JobStatus = "failed"
)

var (
	ErrJobExists    = errors.New("job already exists")
	ErrJobNotFound  = errors.New("job not found")
	ErrInvalidJobID = errors.New("invalid job id")
)

type JobType string

const (
	JobTypeZip     JobType = "zip"
	JobTypeTarball JobType = "tarball"
	JobTypeScript  JobType = "script"
)

type Job struct {
	ID           string
	Type         JobType
	Status       JobStatus
	Progress     int
	CreationTime time.Time
	ExpiresAt    time.Time
	Files        []string
	InputSize    int64
	OutputSize   int64
	Error        string
	Filename     string
}

type Jobs struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewJobs() *Jobs {
	return &Jobs{jobs: make(map[string]*Job)}
}

func (j *Jobs) Add(job Job) error {
	if job.ID == "" {
		return ErrInvalidJobID
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if _, exists := j.jobs[job.ID]; exists {
		return ErrJobExists
	}

	stored := job
	stored.Files = slices.Clone(job.Files)
	j.jobs[job.ID] = &stored
	return nil
}

func (j *Jobs) Get(id string) (Job, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	job, ok := j.jobs[id]
	if !ok {
		return Job{}, false
	}

	result := *job
	result.Files = slices.Clone(job.Files)
	return result, true
}

func (j *Jobs) List() []Job {
	j.mu.RLock()
	defer j.mu.RUnlock()

	out := make([]Job, 0, len(j.jobs))
	for _, job := range j.jobs {
		result := *job
		result.Files = slices.Clone(job.Files)
		out = append(out, result)
	}

	return out
}

func (j *Jobs) MarkProcessing(id string) error {
	return j.update(id, func(job *Job) error {
		job.Status = StatusProcessing
		return nil
	})
}

func (j *Jobs) SetProgress(id string, progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	return j.update(id, func(job *Job) error {
		job.Progress = progress
		return nil
	})
}

func (j *Jobs) MarkFailed(id string, err error) error {
	return j.update(id, func(job *Job) error {
		job.Status = StatusFailed
		job.Error = err.Error()
		return nil
	})
}

func (j *Jobs) MarkDone(id, filename string, outputSize int64) error {
	return j.update(id, func(job *Job) error {
		job.Status = StatusDone
		job.Filename = filename
		job.OutputSize = outputSize
		job.Error = ""
		return nil
	})
}

func (j *Jobs) update(id string, fn func(*Job) error) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	job, ok := j.jobs[id]
	if !ok {
		return ErrJobNotFound
	}
	return fn(job)
}

func (j *Jobs) Delete(id string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	delete(j.jobs, id)
}

func (j *Jobs) Expired(now time.Time) []Job {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var out []Job
	for _, job := range j.jobs {
		if now.After(job.ExpiresAt) {
			result := *job
			result.Files = slices.Clone(job.Files)
			out = append(out, result)
		}
	}
	return out
}
