package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
)

func (m *Manager) executeZipJob(jobID string) error {
	job, err := m.getJobOfType(jobID, JobTypeZip)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".zip"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := createZipFromRoot(outPath, m.sourceRootDir, job.Files, func(progress int) {
		_ = m.jobs.SetProgress(jobID, progress)
	}); err != nil {
		_ = cleanupFile(outPath)
		_ = m.jobs.MarkFailed(jobID, err)
		return err
	}

	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}

func (m *Manager) executeTarballJob(jobID string) error {
	job, err := m.getJobOfType(jobID, JobTypeTarball)
	if err != nil {
		return err
	}

	if err := m.jobs.MarkProcessing(jobID); err != nil {
		return err
	}

	filename := job.ID + ".tar.gz"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := createTarballFromRoot(outPath, m.sourceRootDir, job.Files, func(progress int) {
		_ = m.jobs.SetProgress(jobID, progress)
	}); err != nil {
		_ = cleanupFile(outPath)
		_ = m.jobs.MarkFailed(jobID, err)
		return err
	}

	if err := m.jobs.MarkDone(jobID, filename); err != nil {
		return err
	}

	return nil
}

func createZip(dest string, files []string, onProgress func(int)) error {
	return createZipFromRoot(dest, "", files, onProgress)
}

func createZipFromRoot(dest, sourceRoot string, files []string, onProgress func(int)) error {
	total, err := totalFileSize(archiveSourcePaths(sourceRoot, files))
	if err != nil {
		return fmt.Errorf("calculate zip progress: %w", err)
	}
	reporter := newProgressReporter(total, onProgress)

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, file := range files {
		if err := addFileToZip(zw, archiveSourcePath(sourceRoot, file), file, reporter); err != nil {
			return fmt.Errorf("add %s: %w", file, err)
		}
	}

	if total > 0 && onProgress != nil {
		onProgress(100)
	}

	return nil
}

func addFileToZip(zw *zip.Writer, sourcePath, archivePath string, reporter *progressReporter) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(archivePath)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	return copyWithProgress(w, src, reporter)
}

func createTarball(dest string, files []string, onProgress func(int)) error {
	return createTarballFromRoot(dest, "", files, onProgress)
}

func createTarballFromRoot(dest, sourceRoot string, files []string, onProgress func(int)) error {
	total, err := totalFileSize(archiveSourcePaths(sourceRoot, files))
	if err != nil {
		return fmt.Errorf("calculate tarball progress: %w", err)
	}
	reporter := newProgressReporter(total, onProgress)

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create tarball file: %w", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, file := range files {
		if err := addFileToTarball(tw, archiveSourcePath(sourceRoot, file), file, reporter); err != nil {
			return fmt.Errorf("add %s: %w", file, err)
		}
	}

	if total > 0 && onProgress != nil {
		onProgress(100)
	}

	return nil
}

func addFileToTarball(tw *tar.Writer, sourcePath, archivePath string, reporter *progressReporter) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(archivePath)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	return copyWithProgress(tw, src, reporter)
}

func archiveSourcePaths(sourceRoot string, files []string) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, archiveSourcePath(sourceRoot, file))
	}
	return paths
}

func archiveSourcePath(sourceRoot, file string) string {
	if sourceRoot == "" {
		return file
	}

	return filepath.Join(sourceRoot, file)
}
