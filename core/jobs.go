package core

import (
	"errors"
	"sync"
	"time"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

var (
	ErrJobExists   = errors.New("job already exists")
	ErrJobNotFound = errors.New("job not found")
)

type JobType string

const (
	JobTypeZip     JobType = "zip"
	JobTypeTarball JobType = "tarball"
	JobTypeScript  JobType = "script"
)

type Job struct {
	ID        string    `json:"id"`
	Type      JobType   `json:"type"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	ExpiresAt time.Time `json:"expires_at"`
	Files     []string  `json:"-"`
	Error     string    `json:"error,omitempty"`
	Filename  string    `json:"filename,omitempty"`
}

type Jobs struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewJobs() *Jobs {
	return &Jobs{jobs: make(map[string]*Job)}
}

func (j *Jobs) Add(job *Job) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if _, exists := j.jobs[job.ID]; exists {
		return ErrJobExists
	}

	j.jobs[job.ID] = snapshotJob(job)
	return nil
}

func (j *Jobs) Get(id string) (*Job, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	job, ok := j.jobs[id]
	if !ok {
		return nil, false
	}

	return snapshotJob(job), true
}

func (j *Jobs) Update(id string, fn func(*Job) error) error {
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

func (j *Jobs) Expired(now time.Time) []*Job {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var out []*Job
	for _, job := range j.jobs {
		if now.After(job.ExpiresAt) {
			out = append(out, snapshotJob(job))
		}
	}

	return out
}

func snapshotJob(job *Job) *Job {
	snapshot := *job
	if job.Files != nil {
		snapshot.Files = append([]string(nil), job.Files...)
	}
	return &snapshot
}
