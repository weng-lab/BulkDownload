package service

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
)

const maxJobIDAttempts = 100

type Manager struct {
	jobs            *jobs.Jobs
	jobsDir         string
	sourceRootDir   string
	publicBaseURL   string
	downloadRootDir string
	jobTTL          time.Duration
	generateID      func() string
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

	return &Manager{
		jobs:            jobStore,
		jobsDir:         config.JobsDir,
		sourceRootDir:   config.SourceRootDir,
		publicBaseURL:   config.PublicBaseURL,
		downloadRootDir: config.DownloadRootDir,
		jobTTL:          config.JobTTL,
		generateID:      generateID,
	}
}

func (m *Manager) DispatchZipJob(files []string) (jobs.Job, error) {
	job, err := m.createJob(jobs.JobTypeZip, files)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("create zip job: %w", err)
	}

	go func() {
		logger := slog.Default().With("job_id", job.ID, "job_type", job.Type)
		logger.Info("zip job started")
		if err := m.executeZipJob(job.ID); err != nil {
			logger.Error("zip job failed", "error", err)
			return
		}
		logger.Info("zip job completed")
	}()

	return job, nil
}

func (m *Manager) DispatchTarballJob(files []string) (jobs.Job, error) {
	job, err := m.createJob(jobs.JobTypeTarball, files)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("create tarball job: %w", err)
	}

	go func() {
		logger := slog.Default().With("job_id", job.ID, "job_type", job.Type)
		logger.Info("tarball job started")
		if err := m.executeTarballJob(job.ID); err != nil {
			logger.Error("tarball job failed", "error", err)
			return
		}
		logger.Info("tarball job completed")
	}()

	return job, nil
}

func (m *Manager) DispatchScriptJob(files []string) (jobs.Job, error) {
	job, err := m.createJob(jobs.JobTypeScript, files)
	if err != nil {
		return jobs.Job{}, fmt.Errorf("create script job: %w", err)
	}

	go func() {
		logger := slog.Default().With("job_id", job.ID, "job_type", job.Type)
		logger.Info("script job started")
		if err := m.executeScriptJob(job.ID); err != nil {
			logger.Error("script job failed", "error", err)
			return
		}
		logger.Info("script job completed")
	}()

	return job, nil
}

func (m *Manager) createJob(jobType jobs.JobType, files []string) (jobs.Job, error) {
	expiresAt := time.Now().Add(m.jobTTL)
	job := jobs.Job{
		Type:      jobType,
		Status:    jobs.StatusPending,
		ExpiresAt: expiresAt,
		Files:     append([]string(nil), files...),
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
