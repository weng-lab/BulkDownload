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

func HandleCreateZip(manager *core.Manager, config core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := decodeArchiveFiles(r, config)
		if err != nil {
			http.Error(w, err.Error(), httpStatusForCreateError(err))
			return
		}

		job, err := manager.CreateZipJob(files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("create: zip job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go func() {
			if err := manager.ProcessZipJob(job.ID); err != nil {
				log.Printf("process: zip job %s failed: %v", job.ID, err)
			}
		}()

		writeAcceptedJobResponse(w, job)
	}
}

func HandleCreateTarball(manager *core.Manager, config core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := decodeArchiveFiles(r, config)
		if err != nil {
			http.Error(w, err.Error(), httpStatusForCreateError(err))
			return
		}

		job, err := manager.CreateTarballJob(files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("create: tarball job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go func() {
			if err := manager.ProcessTarballJob(job.ID); err != nil {
				log.Printf("process: tarball job %s failed: %v", job.ID, err)
			}
		}()

		writeAcceptedJobResponse(w, job)
	}
}

func HandleCreateScript(manager *core.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := decodeScriptFiles(r)
		if err != nil {
			http.Error(w, err.Error(), httpStatusForCreateError(err))
			return
		}

		job, err := manager.CreateScriptJob(files)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("create: script job %s accepted with %d files, expires at %s", job.ID, len(job.Files), job.ExpiresAt.Format(time.RFC3339))

		go func() {
			if err := manager.ProcessScriptJob(job.ID); err != nil {
				log.Printf("process: script job %s failed: %v", job.ID, err)
			}
		}()

		writeAcceptedJobResponse(w, job)
	}
}

func HandleStatus(jobs *core.Jobs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/status/")
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
		_ = json.NewEncoder(w).Encode(job)
	}
}

func HandleDownload(jobs *core.Jobs, config core.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/download/")
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

func decodeArchiveFiles(r *http.Request, config core.Config) ([]string, error) {
	req, err := decodeJobRequest(r)
	if err != nil {
		return nil, err
	}

	resolvedFiles, err := resolveArchiveFiles(req.Files, config.SourceRootDir)
	if err != nil {
		return nil, err
	}

	return resolvedFiles, nil
}

func decodeScriptFiles(r *http.Request) ([]string, error) {
	req, err := decodeJobRequest(r)
	if err != nil {
		return nil, err
	}

	normalized := make([]string, 0, len(req.Files))
	for _, file := range req.Files {
		relPath, err := normalizeRelativePath(file)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, relPath)
	}

	return normalized, nil
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

func httpStatusForCreateError(err error) int {
	switch err.Error() {
	case "invalid request body", "files list is empty":
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
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

func resolveArchivePath(raw, sourceRootDir string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	if sourceRootDir == "" {
		if filepath.IsAbs(trimmed) {
			return "", fmt.Errorf("file path must be relative: %s", raw)
		}
		return trimmed, nil
	}

	root := filepath.Clean(sourceRootDir)
	if filepath.IsAbs(trimmed) {
		cleaned := filepath.Clean(trimmed)
		rel, err := filepath.Rel(root, cleaned)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("file path cannot escape source root: %s", raw)
		}
		return rel, nil
	}

	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("file path cannot escape source root: %s", raw)
	}

	return cleaned, nil
}

func resolveArchiveFiles(files []string, sourceRootDir string) ([]string, error) {
	resolved := make([]string, 0, len(files))
	for _, file := range files {
		path, err := resolveArchivePath(file, sourceRootDir)
		if err != nil {
			return nil, err
		}
		checkPath := path
		if sourceRootDir != "" {
			checkPath = filepath.Join(sourceRootDir, path)
		}
		if _, err := os.Stat(checkPath); err != nil {
			return nil, fmt.Errorf("file not found: %s", file)
		}
		resolved = append(resolved, path)
	}

	return resolved, nil
}
