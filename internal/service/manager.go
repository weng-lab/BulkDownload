package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jair/bulkdownload/internal/artifacts"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
)

const maxJobIDAttempts = 100

const maxConcurrentJobs = 4

var ErrDeleteJobRunning = errors.New("job is still running")

type Manager struct {
	jobs            *jobs.Jobs
	jobsDir         string
	sourceRootDir   string
	publicBaseURL   string
	downloadRootDir string
	jobTTL          time.Duration
	generateID      func() string
	sem             chan struct{}
	jobRunsMu       sync.Mutex
	jobRuns         map[string]*jobRun
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type jobRun struct {
	cancel context.CancelFunc
	done   chan struct{}
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

func NewManager(jobStore *jobs.Jobs, config appconfig.Config) *Manager {
	return newManager(jobStore, config, nil)
}

func newManager(jobStore *jobs.Jobs, config appconfig.Config, generateID func() string) *Manager {
	if generateID == nil {
		generateID = randomJobID
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		jobs:            jobStore,
		jobsDir:         config.JobsDir,
		sourceRootDir:   config.SourceRootDir,
		publicBaseURL:   config.PublicBaseURL,
		downloadRootDir: config.DownloadRootDir,
		jobTTL:          config.JobTTL,
		generateID:      generateID,
		sem:             make(chan struct{}, maxConcurrentJobs),
		jobRuns:         make(map[string]*jobRun),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (m *Manager) dispatchJob(job jobs.Job) {
	ctx, cancel := context.WithCancel(m.ctx)
	run := &jobRun{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	m.jobRunsMu.Lock()
	m.jobRuns[job.ID] = run
	m.jobRunsMu.Unlock()

	m.wg.Add(1)
	go func() {
		defer func() {
			m.wg.Done()
			close(run.done)
			m.jobRunsMu.Lock()
			delete(m.jobRuns, job.ID)
			m.jobRunsMu.Unlock()
		}()

		logger := slog.Default().With("job_id", job.ID, "job_type", job.Type)

		select {
		case m.sem <- struct{}{}:
			defer func() {
				<-m.sem
			}()
		case <-ctx.Done():
			logger.Info("job cancelled before start")
			return
		}

		if err := ctx.Err(); err != nil {
			logger.Info("job cancelled before start")
			return
		}

		logger.Info("job started")

		var err error
		switch job.Type {
		case jobs.JobTypeZip:
			err = m.executeZipJob(ctx, job.ID)
		case jobs.JobTypeTarball:
			err = m.executeTarballJob(ctx, job.ID)
		case jobs.JobTypeScript:
			err = m.executeScriptJob(ctx, job.ID)
		default:
			err = fmt.Errorf("unsupported job type %q", job.Type)
		}
		if err != nil {
			logger.Error("job failed", "error", err)
			return
		}

		logger.Info("job completed")
	}()
}

func (m *Manager) Shutdown() {
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) createJob(jobType jobs.JobType, files []string, inputSize int64) (jobs.Job, error) {
	createdAt := time.Now()
	expiresAt := createdAt.Add(m.jobTTL)
	job := jobs.Job{
		Type:      jobType,
		Status:    jobs.StatusPending,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		Files:     append([]string(nil), files...),
		InputSize: inputSize,
	}

	for range maxJobIDAttempts {
		job.ID = m.generateID()

		if err := m.jobs.Add(job); err != nil {
			if errors.Is(err, jobs.ErrJobExists) {
				continue
			}
			return jobs.Job{}, err
		}

		return job, nil
	}

	return jobs.Job{}, errors.New("generate job id: exhausted retries")
}

func outputFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat output file %q: %w", path, err)
	}

	return info.Size(), nil
}

func (m *Manager) GetJobOfType(jobID string, jobType jobs.JobType) (*jobs.Job, error) {
	job, ok := m.jobs.Get(jobID)
	if !ok {
		return nil, jobs.ErrJobNotFound
	}
	if job.Type != jobType {
		return nil, fmt.Errorf("job %s has type %s, not %s", jobID, job.Type, jobType)
	}
	return &job, nil
}

func (m *Manager) DeleteJob(jobID string) error {
	job, ok := m.jobs.Get(jobID)
	// Expired jobs are treated as not found here because the cleanup sweep will remove them shortly.
	if !ok || time.Now().After(job.ExpiresAt) {
		return jobs.ErrJobNotFound
	}

	switch job.Status {
	case jobs.StatusPending, jobs.StatusProcessing:
		m.jobRunsMu.Lock()
		run := m.jobRuns[jobID]
		m.jobRunsMu.Unlock()
		if run != nil {
			run.cancel()
			<-run.done
		}
	}

	job, ok = m.jobs.Get(jobID)
	if !ok {
		return nil
	}

	if filename := artifactFilename(job); filename != "" {
		path := filepath.Join(m.jobsDir, filename)
		if err := artifacts.CleanupFile(path); err != nil {
			return fmt.Errorf("cleanup job artifact %q: %w", filename, err)
		}
	}

	m.jobs.Delete(jobID)
	return nil
}

func artifactFilename(job jobs.Job) string {
	if job.Filename != "" {
		return job.Filename
	}

	switch job.Type {
	case jobs.JobTypeZip:
		return job.ID + ".zip"
	case jobs.JobTypeTarball:
		return job.ID + ".tar.gz"
	case jobs.JobTypeScript:
		return job.ID + ".sh"
	default:
		return ""
	}
}

func randomJobID() string {
	first := randomWord(jobIDWords)
	second := randomWord(jobIDWords)
	for second == first {
		second = randomWord(jobIDWords)
	}
	return first + "-" + second
}

func randomWord(words []string) string {
	return words[rand.IntN(len(words))]
}
