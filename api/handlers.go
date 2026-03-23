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

func HandleCreateJob(manager *core.Manager, config core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeCreateJobRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		jobType, err := parseJobType(req.Type)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		files, err := resolveJobFiles(req.Files, config.SourceRootDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var job core.Job
		switch jobType {
		case core.JobTypeZip:
			job, err = manager.DispatchZipJob(files)
		case core.JobTypeTarball:
			job, err = manager.DispatchTarballJob(files)
		case core.JobTypeScript:
			job, err = manager.DispatchScriptJob(files)
		default:
			http.Error(w, fmt.Sprintf("invalid job type: %s", jobType), http.StatusBadRequest)
			return
		}
		if err != nil {
			log.Printf("create: failed to dispatch %s job: %v", jobType, err)
			http.Error(w, "failed to dispatch job", http.StatusInternalServerError)
			return
		}

		log.Printf("create: %s job %s accepted with %d files, expires at %s", jobType, job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		writeAcceptedJobResponse(w, job)
	}
}

func parseJobType(raw string) (core.JobType, error) {
	jobType := core.JobType(raw)
	switch jobType {
	case core.JobTypeZip, core.JobTypeTarball, core.JobTypeScript:
		return jobType, nil
	default:
		return "", fmt.Errorf("invalid job type: %s", raw)
	}
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

func writeAcceptedJobResponse(w http.ResponseWriter, job core.Job) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(JobResponse{
		ID:        job.ID,
		ExpiresAt: job.ExpiresAt,
	})
}

func decodeCreateJobRequest(r *http.Request) (CreateJobRequest, error) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return CreateJobRequest{}, fmt.Errorf("invalid request body")
	}
	if req.Type == "" {
		return CreateJobRequest{}, fmt.Errorf("type is required")
	}
	if len(req.Files) == 0 {
		return CreateJobRequest{}, fmt.Errorf("files list is empty")
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

		if filepath.IsAbs(file) {
			return nil, fmt.Errorf("absolute paths are not allowed: %s", file)
		}

		checkPath := filepath.Join(sourceRootDir, file)
		if _, err := os.Stat(checkPath); err != nil {
			return nil, fmt.Errorf("file not found: %s", file)
		}
		resolved = append(resolved, file)
	}

	return resolved, nil
}
