package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jair/bulkdownload/api"
	"github.com/jair/bulkdownload/core"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
)

func main() {
	config, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		log.Fatalf("create jobs dir: %v", err)
	}

	jobStore := jobs.NewJobs()
	manager := core.NewManager(jobStore, config)

	core.StartCleanup(jobStore, config.JobsDir, config.CleanupTick)

	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Post("/jobs", api.HandleCreateJob(manager, config))
	r.Get("/status/{id}", api.HandleStatus(jobStore))
	r.Get("/download/{id}", api.HandleDownload(jobStore, config))

	log.Printf(
		"config: jobs_dir=%s source_root_dir=%s public_base_url=%s download_root_dir=%s port=%s job_ttl=%s cleanup_tick=%s",
		config.JobsDir,
		config.SourceRootDir,
		config.PublicBaseURL,
		config.DownloadRootDir,
		config.Port,
		config.JobTTL,
		config.CleanupTick,
	)
	log.Printf("bulk download service listening on :%s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, r))
}
