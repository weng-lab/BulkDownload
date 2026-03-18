package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jair/bulkdownload/api"
	"github.com/jair/bulkdownload/core"
)

func main() {
	core.LoadConfig()

	if err := os.MkdirAll(core.JobsDir, 0o755); err != nil {
		log.Fatalf("failed to create jobs dir: %v", err)
	}

	store := core.NewStore()
	core.StartCleanup(store)

	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Post("/zip", api.HandleCreateZip(store))
	r.Post("/tarball", api.HandleCreateTarball(store))
	r.Post("/script", api.HandleCreateScript(store))
	r.Get("/status/{id}", api.HandleStatus(store))
	r.Get("/download/{id}", api.HandleDownload(store))

	log.Printf("config: jobs_dir=%s source_root_dir=%s public_base_url=%s download_root_dir=%s port=%s zip_ttl=%s cleanup_tick=%s processing_delay=%s", core.JobsDir, core.SourceRootDir, core.PublicBaseURL, core.DownloadRootDir, core.Port, core.ZipTTL, core.CleanupTick, core.ProcessingDelay)
	log.Printf("bulk download service listening on :%s", core.Port)
	log.Fatal(http.ListenAndServe(":"+core.Port, r))
}
