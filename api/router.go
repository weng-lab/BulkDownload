package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

func NewRouter(manager *service.Manager, jobStore *jobs.Jobs, config appconfig.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Post("/jobs", HandleCreateJob(manager, config))
	r.Get("/status/{id}", HandleStatus(jobStore))
	r.Get("/download/{id}", HandleDownload(jobStore, config))

	return r
}
