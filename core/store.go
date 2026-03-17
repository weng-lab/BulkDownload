package core

import (
	"errors"
	"math/rand/v2"
	"sync"
	"time"
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

var jobIDWords = []string{
	"allele",
	"amplic",
	"codon",
	"contig",
	"crispr",
	"exome",
	"exon",
	"genome",
	"intron",
	"kmer",
	"locus",
	"motif",
	"operon",
	"orf",
	"plasmid",
	"primer",
	"repeat",
	"insert",
	"splice",
	"strand",
	"telomer",
	"transpo",
	"utr",
	"utr3",
	"atac",
	"bam",
	"bed",
	"bigwig",
	"cram",
	"chipseq",
	"cpg",
	"dnase",
	"eqtl",
	"fastq",
	"gtf",
	"hic",
	"indel",
	"mirna",
	"peaks",
	"phased",
	"reads",
	"rnaseq",
	"rpkm",
	"snp",
	"sqtl",
	"tad",
	"tpm",
	"vcf",
	"wig",
}

var generateJobID = func() string {
	first := randomWord(jobIDWords)
	second := randomWord(jobIDWords)
	for second == first {
		second = randomWord(jobIDWords)
	}
	return first + "-" + second
}

var jobIDGenerationTimeout = 3 * time.Second

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) CreateJob(files []string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deadline := time.Now().Add(jobIDGenerationTimeout)
	for time.Now().Before(deadline) {
		id := generateJobID()
		if _, exists := s.jobs[id]; exists {
			continue
		}

		job := &Job{
			ID:        id,
			Status:    StatusPending,
			ExpiresAt: time.Now().Add(ZipTTL),
			Files:     files,
		}
		s.jobs[job.ID] = job

		return job, nil
	}

	return nil, errors.New("generate job id: timed out finding unique id")
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

func randomWord(words []string) string {
	return words[rand.IntN(len(words))]
}
