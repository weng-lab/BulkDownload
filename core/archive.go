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
	if err := createZipFromRoot(outPath, m.sourceRootDir, job.Files, func(progress int) {
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
	if err := createTarballFromRoot(outPath, m.sourceRootDir, job.Files, func(progress int) {
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
	return createZipFromRoot(dest, "", files, onProgress)
}

func createZipFromRoot(dest, sourceRoot string, files []string, onProgress func(int)) error {
	if err := validateArchiveFileList(files); err != nil {
		return fmt.Errorf("validate zip inputs: %w", err)
	}

	inputs := archiveInputs(sourceRoot, files)

	total, err := totalFileSize(inputs.sourcePaths())
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

	for _, input := range inputs {
		if err := addFileToZip(zw, input, reporter); err != nil {
			return fmt.Errorf("add %s: %w", input.archivePath, err)
		}
	}

	if total > 0 && onProgress != nil {
		onProgress(100)
	}

	return nil
}

func addFileToZip(zw *zip.Writer, input archiveInput, reporter *progressReporter) error {
	src, err := os.Open(input.sourcePath)
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
	header.Name = filepath.ToSlash(input.archivePath)
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
	if err := validateArchiveFileList(files); err != nil {
		return fmt.Errorf("validate tarball inputs: %w", err)
	}

	inputs := archiveInputs(sourceRoot, files)

	total, err := totalFileSize(inputs.sourcePaths())
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

	for _, input := range inputs {
		if err := addFileToTarball(tw, input, reporter); err != nil {
			return fmt.Errorf("add %s: %w", input.archivePath, err)
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

func addFileToTarball(tw *tar.Writer, input archiveInput, reporter *progressReporter) error {
	return addArchiveFileToTarball(tw, input, reporter)
}

type archiveInput struct {
	sourcePath  string
	archivePath string
}

type archiveInputList []archiveInput

func archiveInputs(sourceRoot string, files []string) archiveInputList {
	inputs := make(archiveInputList, 0, len(files))
	for _, file := range files {
		input := archiveInput{
			sourcePath:  file,
			archivePath: file,
		}
		if sourceRoot != "" {
			input.sourcePath = filepath.Join(sourceRoot, file)
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func (inputs archiveInputList) sourcePaths() []string {
	paths := make([]string, 0, len(inputs))
	for _, input := range inputs {
		paths = append(paths, input.sourcePath)
	}
	return paths
}

func addArchiveFileToTarball(tw *tar.Writer, input archiveInput, reporter *progressReporter) error {
	src, err := os.Open(input.sourcePath)
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
	header.Name = filepath.ToSlash(input.archivePath)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	return copyWithProgress(tw, src, reporter)
}
