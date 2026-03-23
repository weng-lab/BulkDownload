package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

func HandleCreateJob(manager *service.Manager, _ appconfig.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeCreateJobRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		job, err := manager.CreateJob(req.Type, req.Files)
		if err != nil {
			if service.IsCreateJobRequestError(err) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			log.Printf("create: failed to dispatch %s job: %v", req.Type, err)
			http.Error(w, "failed to dispatch job", http.StatusInternalServerError)
			return
		}

		log.Printf("create: %s job %s accepted with %d files, expires at %s", job.Type, job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		writeAcceptedJobResponse(w, job)
	}
}

func HandleStatus(jobStore *jobs.Jobs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobStore.Get(id)
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

func HandleDownload(jobStore *jobs.Jobs, config appconfig.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobStore.Get(id)
		if !ok {
			log.Printf("download: job %s not found", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		if job.Status != jobs.StatusDone {
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

func writeAcceptedJobResponse(w http.ResponseWriter, job jobs.Job) {
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
