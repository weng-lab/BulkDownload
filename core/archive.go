package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
)

func (m *Manager) ProcessZipJob(jobID string) error {
	job, err := m.getJobOfType(jobID, JobTypeZip)
	if err != nil {
		return err
	}

	if err := m.setStatus(jobID, StatusProcessing); err != nil {
		return err
	}

	filename := job.ID + ".zip"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := createZip(outPath, job.Files, func(progress int) {
		_ = m.setProgress(jobID, progress)
	}); err != nil {
		_ = cleanupFile(outPath)
		_ = m.setFailed(jobID, err)
		return err
	}

	if err := m.setDone(jobID, filename); err != nil {
		return err
	}

	return nil
}

func (m *Manager) ProcessTarballJob(jobID string) error {
	job, err := m.getJobOfType(jobID, JobTypeTarball)
	if err != nil {
		return err
	}

	if err := m.setStatus(jobID, StatusProcessing); err != nil {
		return err
	}

	filename := job.ID + ".tar.gz"
	outPath := filepath.Join(m.jobsDir, filename)
	if err := createTarball(outPath, job.Files, func(progress int) {
		_ = m.setProgress(jobID, progress)
	}); err != nil {
		_ = cleanupFile(outPath)
		_ = m.setFailed(jobID, err)
		return err
	}

	if err := m.setDone(jobID, filename); err != nil {
		return err
	}

	return nil
}

func createZip(dest string, files []string, onProgress func(int)) error {
	if err := validateArchiveFileList(files); err != nil {
		return fmt.Errorf("validate zip inputs: %w", err)
	}

	total, err := totalFileSize(files)
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

	for _, path := range files {
		if err := addFileToZip(zw, path, reporter); err != nil {
			return fmt.Errorf("add %s: %w", path, err)
		}
	}

	if total > 0 && onProgress != nil {
		onProgress(100)
	}

	return nil
}

func addFileToZip(zw *zip.Writer, path string, reporter *progressReporter) error {
	src, err := os.Open(path)
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
	header.Name = filepath.ToSlash(path)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	return copyWithProgress(w, src, reporter)
}

func createTarball(dest string, files []string, onProgress func(int)) error {
	if err := validateArchiveFileList(files); err != nil {
		return fmt.Errorf("validate tarball inputs: %w", err)
	}

	total, err := totalFileSize(files)
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

	for _, path := range files {
		if err := addFileToTarball(tw, path, reporter); err != nil {
			return fmt.Errorf("add %s: %w", path, err)
		}
	}

	if total > 0 && onProgress != nil {
		onProgress(100)
	}

	return nil
}

func validateArchiveFileList(files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files provided")
	}

	for _, path := range files {
		if filepath.IsAbs(path) {
			return fmt.Errorf("absolute paths are not supported: %s", path)
		}
	}

	return nil
}

func addFileToTarball(tw *tar.Writer, path string, reporter *progressReporter) error {
	src, err := os.Open(path)
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
	header.Name = filepath.ToSlash(path)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	return copyWithProgress(tw, src, reporter)
}
