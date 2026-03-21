package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type archiveCreator func(string, []string, func(int)) error

type archiveProcessFunc func(*Manager, string) error

func TestCreateArchive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		create    archiveCreator
		read      func(*testing.T, string) map[string]string
		destName  string
		makeFiles func(*testing.T, string) []string
		want      map[string]string
		wantErr   bool
		checkProg bool
	}{
		{
			name:     "zip writes flat archive with progress",
			create:   createZip,
			read:     readZipArchive,
			destName: "result.zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
					writeTestFile(t, filepath.Join(root, "nested", "bravo.txt"), "bravo contents"),
				}
			},
			want: map[string]string{
				"alpha.txt": "alpha contents",
				"bravo.txt": "bravo contents",
			},
			checkProg: true,
		},
		{
			name:     "tarball writes flat archive with progress",
			create:   createTarball,
			read:     readTarballArchive,
			destName: "result.tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
					writeTestFile(t, filepath.Join(root, "nested", "bravo.txt"), "bravo contents"),
				}
			},
			want: map[string]string{
				"alpha.txt": "alpha contents",
				"bravo.txt": "bravo contents",
			},
			checkProg: true,
		},
		{
			name:     "zip rejects empty file list",
			create:   createZip,
			destName: "result.zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return nil
			},
			wantErr: true,
		},
		{
			name:     "tarball rejects empty file list",
			create:   createTarball,
			destName: "result.tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return nil
			},
			wantErr: true,
		},
		{
			name:     "zip rejects duplicate basenames",
			create:   createZip,
			destName: "result.zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
			wantErr: true,
		},
		{
			name:     "tarball rejects duplicate basenames",
			create:   createTarball,
			destName: "result.tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			dest := filepath.Join(root, tt.destName)
			files := tt.makeFiles(t, root)

			var progress []int
			err := tt.create(dest, files, func(percent int) {
				progress = append(progress, percent)
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("create archive error = nil, want non-nil")
				}
				assertFileAbsent(t, dest)
				return
			}
			if err != nil {
				t.Fatalf("create archive error = %v", err)
			}

			if diff := cmp.Diff(tt.want, tt.read(t, dest)); diff != "" {
				t.Errorf("archive contents mismatch (-want +got):\n%s", diff)
			}
			if tt.checkProg {
				assertProgressSequence(t, progress)
			}
		})
	}
}

func TestManagerProcessArchiveJob(t *testing.T) {
	tests := []struct {
		name          string
		jobType       JobType
		createJob     func(*Manager, []string) (*Job, error)
		processJob    archiveProcessFunc
		archiveSuffix string
		makeFiles     func(*testing.T, string) []string
		wantErr       bool
	}{
		{
			name:          "zip marks done after creating archive",
			jobType:       JobTypeZip,
			createJob:     (*Manager).CreateZipJob,
			processJob:    (*Manager).ProcessZipJob,
			archiveSuffix: ".zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")}
			},
		},
		{
			name:          "tarball marks done after creating archive",
			jobType:       JobTypeTarball,
			createJob:     (*Manager).CreateTarballJob,
			processJob:    (*Manager).ProcessTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{writeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents")}
			},
		},
		{
			name:          "zip fails for missing file",
			jobType:       JobTypeZip,
			createJob:     (*Manager).CreateZipJob,
			processJob:    (*Manager).ProcessZipJob,
			archiveSuffix: ".zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{filepath.Join(root, "missing-file.txt")}
			},
			wantErr: true,
		},
		{
			name:          "tarball fails for missing file",
			jobType:       JobTypeTarball,
			createJob:     (*Manager).CreateTarballJob,
			processJob:    (*Manager).ProcessTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{filepath.Join(root, "missing-file.txt")}
			},
			wantErr: true,
		},
		{
			name:          "zip fails for empty file list",
			jobType:       JobTypeZip,
			createJob:     (*Manager).CreateZipJob,
			processJob:    (*Manager).ProcessZipJob,
			archiveSuffix: ".zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return nil
			},
			wantErr: true,
		},
		{
			name:          "tarball fails for empty file list",
			jobType:       JobTypeTarball,
			createJob:     (*Manager).CreateTarballJob,
			processJob:    (*Manager).ProcessTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return nil
			},
			wantErr: true,
		},
		{
			name:          "zip fails for duplicate basenames",
			jobType:       JobTypeZip,
			createJob:     (*Manager).CreateZipJob,
			processJob:    (*Manager).ProcessZipJob,
			archiveSuffix: ".zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
			wantErr: true,
		},
		{
			name:          "tarball fails for duplicate basenames",
			jobType:       JobTypeTarball,
			createJob:     (*Manager).CreateTarballJob,
			processJob:    (*Manager).ProcessTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			manager, jobs, config := testManager(t)
			if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", config.JobsDir, err)
			}

			files := tt.makeFiles(t, t.TempDir())
			job, err := tt.createJob(manager, files)
			if err != nil {
				t.Fatalf("Create%vJob() error = %v", tt.jobType, err)
			}

			assertApproxJobTTL(t, job.ExpiresAt, config.JobTTL)
			if diff := cmp.Diff(tt.jobType, job.Type); diff != "" {
				t.Errorf("job type mismatch (-want +got):\n%s", diff)
			}

			err = tt.processJob(manager, job.ID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("process job error = nil, want non-nil")
				}
				assertFailedArchiveJob(t, jobs, job.ID)
				assertFileAbsent(t, filepath.Join(config.JobsDir, job.ID+tt.archiveSuffix))
				return
			}
			if err != nil {
				t.Fatalf("process job error = %v", err)
			}

			got, ok := jobs.Get(job.ID)
			if !ok {
				t.Fatalf("Get(%q) ok = false, want true", job.ID)
			}

			if got.Filename == "" {
				t.Fatalf("processed job filename = %q, want non-empty", got.Filename)
			}
			want := &Job{
				ID:        job.ID,
				Type:      tt.jobType,
				Status:    StatusDone,
				Progress:  100,
				ExpiresAt: job.ExpiresAt,
				Files:     append([]string(nil), files...),
				Filename:  got.Filename,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("processed job mismatch (-want +got):\n%s", diff)
			}
			if _, err := os.Stat(filepath.Join(config.JobsDir, got.Filename)); err != nil {
				t.Fatalf("Stat(%q) error = %v", filepath.Join(config.JobsDir, got.Filename), err)
			}
		})
	}
}

func assertApproxJobTTL(t *testing.T, expiresAt time.Time, wantTTL time.Duration) {
	t.Helper()

	want := time.Now().Add(wantTTL)
	if diff := cmp.Diff(want, expiresAt, cmpopts.EquateApproxTime(time.Second)); diff != "" {
		t.Errorf("job expiration mismatch (-want +got):\n%s", diff)
	}
}

func assertFailedArchiveJob(t *testing.T, jobs *Jobs, jobID string) {
	t.Helper()

	got, ok := jobs.Get(jobID)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", jobID)
	}
	if got.Status != StatusFailed {
		t.Errorf("job status = %q, want %q", got.Status, StatusFailed)
	}
	if got.Progress != 0 {
		t.Errorf("job progress = %d, want 0", got.Progress)
	}
	if got.Error == "" {
		t.Errorf("job error = %q, want non-empty", got.Error)
	}
	if got.Filename != "" {
		t.Errorf("job filename = %q, want empty", got.Filename)
	}
}

func assertProgressSequence(t *testing.T, got []int) {
	t.Helper()

	if len(got) == 0 {
		t.Fatalf("progress updates = nil, want at least one update")
	}
	if got[len(got)-1] != 100 {
		t.Fatalf("final progress = %d, want 100", got[len(got)-1])
	}
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Fatalf("progress updates = %v, want monotonic sequence", got)
		}
	}
	for _, progress := range got {
		if progress < 0 || progress > 100 {
			t.Fatalf("progress updates = %v, want values in [0,100]", got)
		}
	}
}

func assertFileAbsent(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", path, err)
	}
}

func readZipArchive(t *testing.T, path string) map[string]string {
	t.Helper()

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("OpenReader(%q) error = %v", path, err)
	}
	defer reader.Close()

	contents := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("Open(%q) error = %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("ReadAll(%q) error = %v", file.Name, err)
		}
		contents[file.Name] = string(data)
	}

	return contents
}

func readTarballArchive(t *testing.T, path string) map[string]string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", path, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("NewReader(%q) error = %v", path, err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	contents := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("ReadAll(%q) error = %v", header.Name, err)
		}
		contents[header.Name] = string(data)
	}

	return contents
}

func writeTestFile(t *testing.T, path, contents string) string {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	return path
}
