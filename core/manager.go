package core

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

const maxJobIDAttempts = 100

type Manager struct {
	jobs            *Jobs
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

func NewManager(jobs *Jobs, config Config) *Manager {
	return newManager(jobs, config, nil)
}

func newManager(jobs *Jobs, config Config, generateID func() string) *Manager {
	if generateID == nil {
		generateID = randomJobID
	}

	return &Manager{
		jobs:            jobs,
		jobsDir:         config.JobsDir,
		sourceRootDir:   config.SourceRootDir,
		publicBaseURL:   config.PublicBaseURL,
		downloadRootDir: config.DownloadRootDir,
		jobTTL:          config.JobTTL,
		generateID:      generateID,
	}
}

func (m *Manager) CreateZipJob(files []string) (*Job, error) {
	return m.createJob(JobTypeZip, files)
}

func (m *Manager) CreateTarballJob(files []string) (*Job, error) {
	return m.createJob(JobTypeTarball, files)
}

func (m *Manager) CreateScriptJob(files []string) (*Job, error) {
	return m.createJob(JobTypeScript, files)
}

func (m *Manager) createJob(jobType JobType, files []string) (*Job, error) {
	expiresAt := time.Now().Add(m.jobTTL)
	job := Job{
		Type:      jobType,
		Status:    StatusPending,
		ExpiresAt: expiresAt,
		Files:     append([]string(nil), files...),
	}

	for range maxJobIDAttempts {
		job.ID = m.generateID()

		if err := m.jobs.Add(job); err != nil {
			if errors.Is(err, ErrJobExists) {
				continue
			}
			return nil, err
		}

		createdJob, ok := m.jobs.Get(job.ID)
		if !ok {
			return nil, ErrJobNotFound
		}
		return &createdJob, nil
	}

	return nil, errors.New("generate job id: exhausted retries")
}

func (m *Manager) getJobOfType(jobID string, jobType JobType) (*Job, error) {
	job, ok := m.jobs.Get(jobID)
	if !ok {
		return nil, ErrJobNotFound
	}
	if job.Type != jobType {
		return nil, fmt.Errorf("job %s has type %s, not %s", jobID, job.Type, jobType)
	}
	return &job, nil
}

func (m *Manager) setStatus(jobID string, status JobStatus) error {
	return m.jobs.Update(jobID, func(job *Job) error {
		job.Status = status
		if status == StatusPending {
			job.Progress = 0
		}
		return nil
	})
}

func (m *Manager) setProgress(jobID string, progress int) error {
	return m.jobs.Update(jobID, func(job *Job) error {
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		job.Progress = progress
		return nil
	})
}

func (m *Manager) setFailed(jobID string, err error) error {
	return m.jobs.Update(jobID, func(job *Job) error {
		job.Status = StatusFailed
		job.Error = err.Error()
		return nil
	})
}

func (m *Manager) setDone(jobID, filename string) error {
	return m.jobs.Update(jobID, func(job *Job) error {
		job.Status = StatusDone
		job.Progress = 100
		job.Filename = filename
		job.Error = ""
		return nil
	})
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
