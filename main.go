package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	zipTTL      = 24 * time.Hour
	cleanupTick = 10 * time.Minute
	outputDir   = "./zips"
	defaultPort = "8080"
)

// Job status constants.
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

// Job represents a zip job tracked by the service.
type Job struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	Files     []string  `json:"-"`
	Error     string    `json:"error,omitempty"`
	Filename  string    `json:"filename,omitempty"`
}

// Store is an in-memory thread-safe job store.
type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) Set(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
}

func (s *Store) Expired() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	var out []*Job
	for _, j := range s.jobs {
		if now.After(j.ExpiresAt) {
			out = append(out, j)
		}
	}
	return out
}

// --- Handlers ---

type ZipRequest struct {
	Files []string `json:"files"`
}

type ZipResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func handleCreateZip(store *Store) http.HandlerFunc {
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

		// Validate that every path exists before accepting the job.
		for _, f := range req.Files {
			if _, err := os.Stat(f); err != nil {
				http.Error(w, fmt.Sprintf("file not found: %s", f), http.StatusBadRequest)
				return
			}
		}

		job := &Job{
			ID:        uuid.NewString(),
			Status:    StatusPending,
			ExpiresAt: time.Now().Add(zipTTL),
			Files:     req.Files,
		}
		store.Set(job)

		// Kick off the zip in the background.
		go processJob(store, job)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(ZipResponse{
			ID:        job.ID,
			ExpiresAt: job.ExpiresAt,
		})
	}
}

func handleStatus(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/status/")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := store.Get(id)
		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	}
}

func handleDownload(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/download/")
		if id == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		job, ok := store.Get(id)
		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		if job.Status != StatusDone {
			http.Error(w, "zip is not ready yet", http.StatusConflict)
			return
		}

		zipPath := filepath.Join(outputDir, job.Filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, job.Filename))
		http.ServeFile(w, r, zipPath)
	}
}

// --- Zip worker ---

func processJob(store *Store, job *Job) {
	store.mu.Lock()
	job.Status = StatusProcessing
	store.mu.Unlock()

	filename := job.ID + ".zip"
	outPath := filepath.Join(outputDir, filename)

	if err := createZip(outPath, job.Files); err != nil {
		log.Printf("zip failed for job %s: %v", job.ID, err)
		store.mu.Lock()
		job.Status = StatusFailed
		job.Error = err.Error()
		store.mu.Unlock()
		return
	}

	store.mu.Lock()
	job.Status = StatusDone
	job.Filename = filename
	store.mu.Unlock()

	log.Printf("job %s complete: %s", job.ID, outPath)
}

func createZip(dest string, files []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, path := range files {
		if err := addFileToZip(zw, path); err != nil {
			return fmt.Errorf("add %s: %w", path, err)
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, path string) error {
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
	// Use the base name so the zip is flat (no nested dirs).
	header.Name = filepath.Base(path)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}

// --- Cleanup ---

func startCleanup(store *Store) {
	ticker := time.NewTicker(cleanupTick)
	go func() {
		for range ticker.C {
			expired := store.Expired()
			for _, job := range expired {
				if job.Filename != "" {
					path := filepath.Join(outputDir, job.Filename)
					if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
						log.Printf("cleanup: failed to remove %s: %v", path, err)
					} else {
						log.Printf("cleanup: removed %s", path)
					}
				}
				store.Delete(job.ID)
			}
		}
	}()
}

// --- Main ---

func main() {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	store := NewStore()
	startCleanup(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/zip", handleCreateZip(store))
	mux.HandleFunc("/status/", handleStatus(store))
	mux.HandleFunc("/download/", handleDownload(store))

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("bulk download service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
