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
	r.Use(newRequestLoggerMiddleware(logger))
	r.Use(cors.AllowAll().Handler)

	r.Post("/jobs", HandleCreateJob(manager, config))
	r.Get("/status/{id}", HandleStatus(jobStore))
	r.Get("/download/{id}", HandleDownload(jobStore, config))
	r.Get("/admin/jobs", HandleAdminListJobs(jobStore))
	r.Get("/admin/jobs/{id}", HandleAdminGetJob(jobStore))

	return r
}
