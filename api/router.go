package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

func NewRouter(logger *slog.Logger, manager *service.Manager, jobStore *jobs.Jobs, config appconfig.Config) http.Handler {
	r := chi.NewRouter()
	a := chi.NewRouter()

	r.Use(newRequestLoggerMiddleware(logger))
	r.Use(cors.AllowAll().Handler)

	a.Use(adminAuth(config.AdminToken))

	a.Get("/jobs", HandleAdminListJobs(jobStore))
	a.Get("/jobs/{id}", HandleAdminGetJob(jobStore))
	a.Delete("/jobs/{id}", HandleAdminDeleteJob(manager))
	r.Mount("/admin", a)

	r.Post("/jobs", HandleCreateJob(manager, config))
	r.Get("/status/{id}", HandleStatus(jobStore))
	r.Get("/download/{id}", HandleDownload(jobStore, config))

	return r
}
