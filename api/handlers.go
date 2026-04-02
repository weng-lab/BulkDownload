package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

func HandleCreateJob(manager *service.Manager, _ appconfig.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := requestLogger(r)

		req, err := decodeCreateJobRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		job, err := manager.CreateJob(req.Type, req.Files)
		if err != nil {
			writeCreateJobError(w, logger, req.Type, err)
			return
		}

		logger.Info(
			"create job accepted",
			"job_id", job.ID,
			"job_type", job.Type,
			"file_count", len(job.Files),
			"expires_at", job.ExpiresAt.Format(time.RFC3339),
		)

		writeAcceptedJobResponse(w, job)
	}
}

func HandleStatus(jobStore *jobs.Jobs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := requestLogger(r)

		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobStore.Get(id)
		if !ok {
			logger.Info("status job not found", "job_id", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		logger.Info("status returned", "job_id", id, "job_type", job.Type, "job_status", job.Status)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newJobStatusResponse(job))
	}
}

func HandleDownload(jobStore *jobs.Jobs, config appconfig.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := requestLogger(r)

		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobStore.Get(id)
		if !ok {
			logger.Info("download job not found", "job_id", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		if job.Status != jobs.StatusDone {
			logger.Info("download not ready", "job_id", id, "job_type", job.Type, "job_status", job.Status)
			http.Error(w, "download is not ready yet", http.StatusConflict)
			return
		}

		downloadPath := filepath.Join(config.JobsDir, job.Filename)
		logger.Info("download served", "job_id", id, "job_type", job.Type, "filename", job.Filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, job.Filename))
		http.ServeFile(w, r, downloadPath)
	}
}

func HandleAdminListJobs(jobStore *jobs.Jobs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := requestLogger(r)
		now := time.Now()
		allJobs := jobStore.List()
		visible := make([]jobs.Job, 0, len(allJobs))
		for _, job := range allJobs {
			if now.After(job.ExpiresAt) {
				continue
			}
			visible = append(visible, job)
		}
		slices.SortFunc(visible, func(a, b jobs.Job) int {
			return b.CreatedAt.Compare(a.CreatedAt)
		})

		resp := make([]AdminJobResponse, 0, len(visible))
		for _, job := range visible {
			resp = append(resp, newAdminJobResponse(job))
		}

		logger.Info("admin jobs listed", "job_count", len(resp))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleAdminGetJob(jobStore *jobs.Jobs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := requestLogger(r)

		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := jobStore.Get(id)
		// Expired jobs are treated as not found here because the cleanup sweep will remove them shortly.
		if !ok || time.Now().After(job.ExpiresAt) {
			logger.Info("admin job not found", "job_id", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		logger.Info("admin job returned", "job_id", id, "job_type", job.Type, "job_status", job.Status)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newAdminJobResponse(job))
	}
}

func HandleAdminDeleteJob(manager *service.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := requestLogger(r)

		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		err := manager.DeleteJob(id)
		switch {
		case err == nil:
			logger.Info("admin job deleted", "job_id", id)
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, jobs.ErrJobNotFound):
			logger.Info("admin delete job not found", "job_id", id)
			http.Error(w, "job not found", http.StatusNotFound)
		case errors.Is(err, service.ErrDeleteJobRunning):
			logger.Info("admin delete job still running", "job_id", id)
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			logger.Error("admin delete job failed", "job_id", id, "error", err)
			http.Error(w, "failed to delete job", http.StatusInternalServerError)
		}
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

func writeCreateJobError(w http.ResponseWriter, logger *slog.Logger, requestedType string, err error) {
	if errors.Is(err, service.ErrCreateJobRequest) {
		http.Error(w, strings.TrimPrefix(err.Error(), service.ErrCreateJobRequest.Error()+": "), http.StatusBadRequest)
		return
	}

	logger.Error("create job failed", "job_type", requestedType, "error", err)
	http.Error(w, "failed to dispatch job", http.StatusInternalServerError)
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
