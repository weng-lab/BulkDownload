package service

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
	"github.com/jair/bulkdownload/internal/artifacts"
)

type archiveCreator func(string, string, []string, func(int)) error

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
			name:     "zip preserves relative paths with progress",
			create:   artifacts.CreateZipFromRoot,
			read:     readZipArchive,
			destName: "result.zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "nested", "bravo.txt"), "bravo contents"),
				}
			},
			checkProg: true,
		},
		{
			name:     "tarball preserves relative paths with progress",
			create:   artifacts.CreateTarballFromRoot,
			read:     readTarballArchive,
			destName: "result.tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "nested", "bravo.txt"), "bravo contents"),
				}
			},
			checkProg: true,
		},
		{
			name:     "zip allows duplicate basenames in different directories",
			create:   artifacts.CreateZipFromRoot,
			read:     readZipArchive,
			destName: "result.zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
			checkProg: true,
		},
		{
			name:     "tarball allows duplicate basenames in different directories",
			create:   artifacts.CreateTarballFromRoot,
			read:     readTarballArchive,
			destName: "result.tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
			checkProg: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := testArchiveRoot(t)
			dest := filepath.Join(root, tt.destName)
			files := tt.makeFiles(t, root)

			var progress []int
			err := tt.create(dest, "", files, func(percent int) {
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

			wantContents := tt.want
			if wantContents == nil {
				wantContents = make(map[string]string, len(files))
				for _, file := range files {
					data, err := os.ReadFile(file)
					if err != nil {
						t.Fatalf("ReadFile(%q) error = %v", file, err)
					}
					wantContents[filepath.ToSlash(file)] = string(data)
				}
			}

			if diff := cmp.Diff(wantContents, tt.read(t, dest)); diff != "" {
				t.Errorf("archive contents mismatch (-want +got):\n%s", diff)
			}
			if tt.checkProg {
				assertProgressSequence(t, progress)
			}
		})
	}
}

func TestCreateArchive_ReportsOneHundredBeforeReturn(t *testing.T) {
	tests := []struct {
		name     string
		create   archiveCreator
		destName string
	}{
		{
			name:     "zip reports completion before returning",
			create:   artifacts.CreateZipFromRoot,
			destName: "result.zip",
		},
		{
			name:     "tarball reports completion before returning",
			create:   artifacts.CreateTarballFromRoot,
			destName: "result.tar.gz",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			root := testArchiveRoot(t)
			dest := filepath.Join(root, tt.destName)
			files := []string{
				writeRelativeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
				writeRelativeTestFile(t, filepath.Join(root, "nested", "bravo.txt"), "bravo contents"),
			}

			reported := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error, 1)

			go func() {
				done <- tt.create(dest, "", files, func(progress int) {
					if progress != 100 {
						return
					}

					select {
					case <-reported:
					default:
						close(reported)
					}

					<-release
				})
			}()

			select {
			case <-reported:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for progress 100 update")
			}

			select {
			case err := <-done:
				t.Fatalf("create archive returned before progress callback released: %v", err)
			default:
			}

			close(release)

			if err := <-done; err != nil {
				t.Fatalf("create archive error = %v", err)
			}
		})
	}
}

func TestManagerExecuteArchiveJob(t *testing.T) {
	tests := []struct {
		name          string
		jobType       JobType
		executeJob    func(*Manager, string) error
		archiveSuffix string
		makeFiles     func(*testing.T, string) []string
		wantErr       bool
	}{
		{
			name:          "zip marks done after creating archive",
			jobType:       JobTypeZip,
			executeJob:    (*Manager).executeZipJob,
			archiveSuffix: ".zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "nested", "shared.txt"), "bravo contents"),
				}
			},
		},
		{
			name:          "tarball marks done after creating archive",
			jobType:       JobTypeTarball,
			executeJob:    (*Manager).executeTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "alpha.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "nested", "shared.txt"), "bravo contents"),
				}
			},
		},
		{
			name:          "zip fails for missing file",
			jobType:       JobTypeZip,
			executeJob:    (*Manager).executeZipJob,
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
			executeJob:    (*Manager).executeTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{filepath.Join(root, "missing-file.txt")}
			},
			wantErr: true,
		},
		{
			name:          "zip allows duplicate basenames in different directories",
			jobType:       JobTypeZip,
			executeJob:    (*Manager).executeZipJob,
			archiveSuffix: ".zip",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
		},
		{
			name:          "tarball allows duplicate basenames in different directories",
			jobType:       JobTypeTarball,
			executeJob:    (*Manager).executeTarballJob,
			archiveSuffix: ".tar.gz",
			makeFiles: func(t *testing.T, root string) []string {
				t.Helper()
				return []string{
					writeRelativeTestFile(t, filepath.Join(root, "first", "shared.txt"), "alpha contents"),
					writeRelativeTestFile(t, filepath.Join(root, "second", "shared.txt"), "bravo contents"),
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fixture := newTestFixture(t)
			if err := os.MkdirAll(fixture.config.JobsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", fixture.config.JobsDir, err)
			}

			files := tt.makeFiles(t, testArchiveRoot(t))
			job, err := fixture.manager.createJob(tt.jobType, files)
			if err != nil {
				t.Fatalf("createJob(%v) error = %v", tt.jobType, err)
			}

			assertApproxJobTTL(t, job.ExpiresAt, fixture.config.JobTTL)
			if diff := cmp.Diff(tt.jobType, job.Type); diff != "" {
				t.Errorf("job type mismatch (-want +got):\n%s", diff)
			}

			err = tt.executeJob(fixture.manager, job.ID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("execute job error = nil, want non-nil")
				}
				assertFailedArchiveJob(t, fixture.jobs, job.ID)
				assertFileAbsent(t, filepath.Join(fixture.config.JobsDir, job.ID+tt.archiveSuffix))
				return
			}
			if err != nil {
				t.Fatalf("execute job error = %v", err)
			}

			got, ok := fixture.jobs.Get(job.ID)
			if !ok {
				t.Fatalf("Get(%q) ok = false, want true", job.ID)
			}

			if got.Filename == "" {
				t.Fatalf("processed job filename = %q, want non-empty", got.Filename)
			}
			want := Job{
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
			archivePath := filepath.Join(fixture.config.JobsDir, got.Filename)
			if _, err := os.Stat(archivePath); err != nil {
				t.Fatalf("Stat(%q) error = %v", archivePath, err)
			}

			wantContents := make(map[string]string, len(files))
			for _, file := range files {
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatalf("ReadFile(%q) error = %v", file, err)
				}
				wantContents[filepath.ToSlash(file)] = string(data)
			}

			gotContents := readArchiveByType(t, archivePath, tt.jobType)
			if diff := cmp.Diff(wantContents, gotContents); diff != "" {
				t.Errorf("archive contents mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func readArchiveByType(t *testing.T, path string, jobType JobType) map[string]string {
	t.Helper()

	if jobType == JobTypeZip {
		return readZipArchive(t, path)
	}

	return readTarballArchive(t, path)
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

func writeRelativeTestFile(t *testing.T, path, contents string) string {
	t.Helper()

	absPath := writeTestFile(t, path, contents)
	relPath, err := filepath.Rel(".", absPath)
	if err != nil {
		t.Fatalf("Rel(%q) error = %v", absPath, err)
	}

	return relPath
}

func writeAbsoluteTestFile(t *testing.T, path, contents string) string {
	t.Helper()

	relPath := writeTestFile(t, path, contents)
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		t.Fatalf("Abs(%q) error = %v", relPath, err)
	}

	return absPath
}

func testArchiveRoot(t *testing.T) string {
	t.Helper()

	root, err := os.MkdirTemp(".", "archive-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Fatalf("RemoveAll(%q) error = %v", root, err)
		}
	})

	return root
}
