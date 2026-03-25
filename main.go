package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jair/bulkdownload/api"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

const (
	httpServerReadHeaderTimeout = 5 * time.Second
	httpServerReadTimeout       = 30 * time.Second
	httpServerIdleTimeout       = 2 * time.Minute
)

func newRouter(manager *service.Manager, jobStore *jobs.Jobs, config appconfig.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Post("/jobs", api.HandleCreateJob(manager, config))
	r.Get("/status/{id}", api.HandleStatus(jobStore))
	r.Get("/download/{id}", api.HandleDownload(jobStore, config))

	return r
}

func newHTTPServer(config appconfig.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + config.Port,
		Handler:           handler,
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
		ReadTimeout:       httpServerReadTimeout,
		IdleTimeout:       httpServerIdleTimeout,
	}
}

func main() {
	config, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		log.Fatalf("create jobs dir: %v", err)
	}

	jobStore := jobs.NewJobs()
	manager := service.NewManager(jobStore, config)

	stopCleanup := service.StartCleanup(jobStore, config.JobsDir, config.CleanupTick)
	defer stopCleanup()

	server := newHTTPServer(config, newRouter(manager, jobStore, config))

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
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
