package core

import (
	"errors"
	"fmt"
	"log"
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

func (m *Manager) DispatchZipJob(files []string) (Job, error) {
	job, err := m.createJob(JobTypeZip, files)
	if err != nil {
		return Job{}, fmt.Errorf("create zip job: %w", err)
	}

	go func() {
		if err := m.executeZipJob(job.ID); err != nil {
			log.Printf("dispatch: zip job %s failed: %v", job.ID, err)
		}
	}()

	return job, nil
}

func (m *Manager) DispatchTarballJob(files []string) (Job, error) {
	job, err := m.createJob(JobTypeTarball, files)
	if err != nil {
		return Job{}, fmt.Errorf("create tarball job: %w", err)
	}

	go func() {
		if err := m.executeTarballJob(job.ID); err != nil {
			log.Printf("dispatch: tarball job %s failed: %v", job.ID, err)
		}
	}()

	return job, nil
}

func (m *Manager) DispatchScriptJob(files []string) (Job, error) {
	job, err := m.createJob(JobTypeScript, files)
	if err != nil {
		return Job{}, fmt.Errorf("create script job: %w", err)
	}

	go func() {
		if err := m.executeScriptJob(job.ID); err != nil {
			log.Printf("dispatch: script job %s failed: %v", job.ID, err)
		}
	}()

	return job, nil
}

func (m *Manager) createJob(jobType JobType, files []string) (Job, error) {
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
			return Job{}, err
		}

		return job, nil
	}

	return Job{}, errors.New("generate job id: exhausted retries")
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
