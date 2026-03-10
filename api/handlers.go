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

	"github.com/jair/bulkdownload/core"
)

type ZipRequest struct {
	Files []string `json:"files"`
}

type ZipResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func HandleCreateZip(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ZipRequest
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

		job := core.NewJob(req.Files)
		store.Set(job)
		log.Printf("create: job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go core.ProcessJob(store, job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(ZipResponse{
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
			http.Error(w, "zip is not ready yet", http.StatusConflict)
			return
		}

		zipPath := filepath.Join(core.OutputDir, job.Filename)
		log.Printf("download: serving job %s from %s", id, zipPath)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, job.Filename))
		http.ServeFile(w, r, zipPath)
	}
}
