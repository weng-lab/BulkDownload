package core

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func ProcessTarballJob(store *Store, job *Job) {
	store.SetStatus(job.ID, StatusProcessing)

	log.Printf("job %s tarball processing started", job.ID)
	time.Sleep(ProcessingDelay)

	filename := job.ID + ".tar.gz"
	outPath := filepath.Join(JobsDir, filename)

	if err := createTarball(outPath, job.Files); err != nil {
		log.Printf("tarball failed for job %s: %v", job.ID, err)
		if removeErr := os.Remove(outPath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("cleanup failed tarball for job %s: %v", job.ID, removeErr)
		}
		store.SetFailed(job.ID, err)
		return
	}

	store.SetDone(job.ID, filename)

	log.Printf("job %s complete: %s", job.ID, outPath)
}

func createTarball(dest string, files []string) error {
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
		if err := addFileToTarball(tw, path); err != nil {
			return fmt.Errorf("add %s: %w", path, err)
		}
	}

	return nil
}

func addFileToTarball(tw *tar.Writer, path string) error {
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
	header.Name = filepath.Base(path)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, src)
	return err
}
