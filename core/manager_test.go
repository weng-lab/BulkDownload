package core

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCreateZipJob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		seedExisting []string
		generatedIDs []string
		files        []string
		wantJob      *Job
		wantErr      string
		wantCalls    int
	}{
		{
			name:         "creates pending job on first available id",
			generatedIDs: []string{"codon-bam"},
			files:        []string{"file.txt"},
			wantJob: &Job{
				ID:       "codon-bam",
				Type:     JobTypeZip,
				Status:   StatusPending,
				Files:    []string{"file.txt"},
				Progress: 0,
			},
			wantCalls: 1,
		},
		{
			name:         "retries duplicate id and keeps requested files",
			seedExisting: []string{"allele-atac"},
			generatedIDs: []string{"allele-atac", "codon-bam"},
			files:        []string{"reads/sample.fastq", "variants/sample.vcf"},
			wantJob: &Job{
				ID:       "codon-bam",
				Type:     JobTypeZip,
				Status:   StatusPending,
				Files:    []string{"reads/sample.fastq", "variants/sample.vcf"},
				Progress: 0,
			},
			wantCalls: 2,
		},
		{
			name:         "stops after max retries when ids always collide",
			seedExisting: []string{"allele-atac"},
			generatedIDs: repeatID("allele-atac", maxJobIDAttempts),
			files:        []string{"file.txt"},
			wantErr:      "generate job id: exhausted retries",
			wantCalls:    maxJobIDAttempts,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testConfig(t)
			jobs := NewJobs()
			for _, id := range tt.seedExisting {
				if err := jobs.Add(Job{ID: id, Type: JobTypeZip, ExpiresAt: time.Unix(100, 0)}); err != nil {
					t.Fatalf("Add(%q) error = %v", id, err)
				}
			}

			calls := 0
			manager := newManager(jobs, config, func() string {
				id := tt.generatedIDs[calls]
				calls++
				return id
			})

			before := time.Now()
			job, err := manager.CreateZipJob(tt.files)
			after := time.Now()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("CreateZipJob() error = nil, want non-nil")
				}
				if diff := cmp.Diff(tt.wantErr, err.Error()); diff != "" {
					t.Errorf("CreateZipJob() error mismatch (-want +got):\n%s", diff)
				}
				if job != nil {
					t.Errorf("CreateZipJob() job = %#v, want nil", job)
				}
			} else {
				if err != nil {
					t.Fatalf("CreateZipJob() error = %v", err)
				}

				want := *tt.wantJob
				want.ExpiresAt = job.ExpiresAt
				if diff := cmp.Diff(want, *job, cmpopts.EquateApproxTime(time.Second)); diff != "" {
					t.Errorf("CreateZipJob() mismatch (-want +got):\n%s", diff)
				}

				wantMin := before.Add(config.JobTTL)
				wantMax := after.Add(config.JobTTL)
				if job.ExpiresAt.Before(wantMin) || job.ExpiresAt.After(wantMax) {
					t.Errorf("job.ExpiresAt = %v, want between %v and %v", job.ExpiresAt, wantMin, wantMax)
				}
			}

			if diff := cmp.Diff(tt.wantCalls, calls); diff != "" {
				t.Errorf("generateID call count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateJob_CopiesRequestedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jobType JobType
		create  func(*Manager, []string) (*Job, error)
	}{
		{
			name:    "zip job",
			jobType: JobTypeZip,
			create:  (*Manager).CreateZipJob,
		},
		{
			name:    "tarball job",
			jobType: JobTypeTarball,
			create:  (*Manager).CreateTarballJob,
		},
		{
			name:    "script job",
			jobType: JobTypeScript,
			create:  (*Manager).CreateScriptJob,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testConfig(t)
			manager := newManager(NewJobs(), config, func() string { return "codon-bam" })
			files := []string{"reads/sample.fastq", "variants/sample.vcf"}

			job, err := tt.create(manager, files)
			if err != nil {
				t.Fatalf("create(%q) error = %v", tt.jobType, err)
			}

			files[0] = "mutated.txt"
			if diff := cmp.Diff([]string{"reads/sample.fastq", "variants/sample.vcf"}, job.Files); diff != "" {
				t.Errorf("job.Files mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.jobType, job.Type); diff != "" {
				t.Errorf("job.Type mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(StatusPending, job.Status); diff != "" {
				t.Errorf("job.Status mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, job.Progress); diff != "" {
				t.Errorf("job.Progress mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateZipJob_ReturnsStoredSnapshot(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	jobs := NewJobs()
	manager := newManager(jobs, config, func() string { return "codon-bam" })

	job, err := manager.CreateZipJob([]string{"file.txt"})
	if err != nil {
		t.Fatalf("CreateZipJob() error = %v", err)
	}

	job.Files[0] = "changed.txt"

	stored, ok := jobs.Get(job.ID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", job.ID)
	}
	if diff := cmp.Diff([]string{"file.txt"}, stored.Files); diff != "" {
		t.Errorf("stored job files mismatch (-want +got):\n%s", diff)
	}
}

func repeatID(id string, count int) []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = id
	}
	return ids
}
