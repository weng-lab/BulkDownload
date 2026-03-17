package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jair/bulkdownload/core"
)

type JobRequest struct {
	Files []string `json:"files"`
}

type JobResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func HandleCreateZip(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req JobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(req.Files) == 0 {
			http.Error(w, "files list is empty", http.StatusBadRequest)
			return
		}

		for _, f := range req.Files {
			if _, err := os.Stat(f); err != nil {
				http.Error(w, fmt.Sprintf("file not found: %s", f), http.StatusBadRequest)
				return
			}
		}

		job, err := store.CreateJob(req.Files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("create: job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go core.ProcessJob(store, job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(JobResponse{
			ID:        job.ID,
			ExpiresAt: job.ExpiresAt,
		})
	}
}

func HandleCreateTarball(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req JobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(req.Files) == 0 {
			http.Error(w, "files list is empty", http.StatusBadRequest)
			return
		}

		for _, f := range req.Files {
			if _, err := os.Stat(f); err != nil {
				http.Error(w, fmt.Sprintf("file not found: %s", f), http.StatusBadRequest)
				return
			}
		}

		job, err := store.CreateJob(req.Files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("tarball create: job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go core.ProcessTarballJob(store, job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(JobResponse{
			ID:        job.ID,
			ExpiresAt: job.ExpiresAt,
		})
	}
}

func HandleCreateScript(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req JobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(req.Files) == 0 {
			http.Error(w, "files list is empty", http.StatusBadRequest)
			return
		}

		normalized := make([]string, 0, len(req.Files))
		for _, file := range req.Files {
			relPath, err := normalizeRelativePath(file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			normalized = append(normalized, relPath)
		}

		job, err := store.CreateJob(normalized)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("script create: job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go core.ProcessScriptJob(store, job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(JobResponse{
			ID:        job.ID,
			ExpiresAt: job.ExpiresAt,
		})
	}
}

func HandleStatus(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/status/")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := store.Get(id)
		if !ok {
			log.Printf("status: job %s not found", id)
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		log.Printf("status: job %s is %s", id, job.Status)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(job)
	}
}

func HandleDownload(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/download/")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := store.Get(id)
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

		zipPath := filepath.Join(core.JobsDir, job.Filename)
		log.Printf("download: serving job %s from %s", id, zipPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, job.Filename))
		http.ServeFile(w, r, zipPath)
	}
}

func normalizeRelativePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("file path must be relative: %s", raw)
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("file path cannot escape the download root: %s", raw)
	}

	return cleaned, nil
}
