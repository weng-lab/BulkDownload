package main

import (
	"sync"
	"time"
)

const (
	zipTTL          = 30 * time.Second
	cleanupTick     = 5 * time.Second
	processingDelay = 5 * time.Second
	outputDir       = "./zips"
	defaultPort     = "8080"
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

func (s *Store) Set(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
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
