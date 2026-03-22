package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jair/bulkdownload/core"
)

func HandleCreateZip(manager *core.Manager, config core.Config) http.HandlerFunc {
	return handleCreateJob(
		"zip",
		config.SourceRootDir,
		manager.CreateZipJob,
		manager.ProcessZipJob,
	)
}

func HandleCreateTarball(manager *core.Manager, config core.Config) http.HandlerFunc {
	return handleCreateJob(
		"tarball",
		config.SourceRootDir,
		manager.CreateTarballJob,
		manager.ProcessTarballJob,
	)
}

func HandleCreateScript(manager *core.Manager, config core.Config) http.HandlerFunc {
	return handleCreateJob(
		"script",
		config.SourceRootDir,
		manager.CreateScriptJob,
		manager.ProcessScriptJob,
	)
}

func HandleStatus(jobs *core.Jobs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobs.Get(id)
		if !ok {
			log.Printf("status: job %s not found", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		log.Printf("status: job %s is %s", id, job.Status)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newJobStatusResponse(job))
	}
}

func HandleDownload(jobs *core.Jobs, config core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobs.Get(id)
		if !ok {
			log.Printf("download: job %s not found", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		if job.Status != core.StatusDone {
			log.Printf("download: job %s not ready, current status %s", id, job.Status)
			http.Error(w, "download is not ready yet", http.StatusConflict)
			return
		}

		downloadPath := filepath.Join(config.JobsDir, job.Filename)
		log.Printf("download: serving job %s from %s", id, downloadPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, job.Filename))
		http.ServeFile(w, r, downloadPath)
	}
}

func writeAcceptedJobResponse(w http.ResponseWriter, job *core.Job) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(JobResponse{
		ID:        job.ID,
		ExpiresAt: job.ExpiresAt,
	})
}

func handleCreateJob(
	jobType string,
	sourceRootDir string,
	createJob func([]string) (*core.Job, error),
	processJob func(string) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeJobRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		files, err := resolveJobFiles(req.Files, sourceRootDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		job, err := createJob(files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("create: %s job %s accepted with %d files, expires at %s", jobType, job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go func() {
			if err := processJob(job.ID); err != nil {
				log.Printf("process: %s job %s failed: %v", jobType, job.ID, err)
			}
		}()

		writeAcceptedJobResponse(w, job)
	}
}
func decodeJobRequest(r *http.Request) (JobRequest, error) {
	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return JobRequest{}, fmt.Errorf("invalid request body")
	}
	if len(req.Files) == 0 {
		return JobRequest{}, fmt.Errorf("files list is empty")
	}

	return req, nil
}

func resolveJobFiles(files []string, sourceRootDir string) ([]string, error) {
	resolved := make([]string, 0, len(files))
	for _, rawPath := range files {
		file := strings.TrimSpace(rawPath)
		if file == "" {
			return nil, fmt.Errorf("file path cannot be empty")
		}

		checkPath := filepath.Join(sourceRootDir, file)
		if _, err := os.Stat(checkPath); err != nil {
			return nil, fmt.Errorf("file not found: %s", file)
		}
		resolved = append(resolved, file)
	}

	return resolved, nil
}
