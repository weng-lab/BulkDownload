package core

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	Files     []string  `json:"-"`
	Error     string    `json:"error,omitempty"`
	Filename  string    `json:"filename,omitempty"`
}

type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func NewJob(files []string) *Job {
	return &Job{
		ID:        uuid.NewString(),
		Status:    StatusPending,
		ExpiresAt: time.Now().Add(ZipTTL),
		Files:     files,
	}
}

func (s *Store) Set(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

func (s *Store) SetStatus(id, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return false
	}

	job.Status = status
	return true
}

func (s *Store) SetFailed(id string, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return false
	}

	job.Status = StatusFailed
	job.Error = err.Error()
	return true
}

func (s *Store) SetDone(id, filename string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return false
	}

	job.Status = StatusDone
	job.Filename = filename
	job.Error = ""
	return true
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
}

func (s *Store) Expired() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var out []*Job
	for _, j := range s.jobs {
		if now.After(j.ExpiresAt) {
			out = append(out, j)
		}
	}

	return out
}
