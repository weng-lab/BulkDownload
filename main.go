package main

import (
	"log"
	"net/http"
	"os"

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

	mux := http.NewServeMux()
	mux.HandleFunc("/zip", api.HandleCreateZip(store))
	mux.HandleFunc("/tarball", api.HandleCreateTarball(store))
	mux.HandleFunc("/script", api.HandleCreateScript(store))
	mux.HandleFunc("/status/", api.HandleStatus(store))
	mux.HandleFunc("/download/", api.HandleDownload(store))

	log.Printf("config: jobs_dir=%s source_root_dir=%s public_base_url=%s download_root_dir=%s port=%s zip_ttl=%s cleanup_tick=%s processing_delay=%s", core.JobsDir, core.SourceRootDir, core.PublicBaseURL, core.DownloadRootDir, core.Port, core.ZipTTL, core.CleanupTick, core.ProcessingDelay)
	log.Printf("bulk download service listening on :%s", core.Port)
	log.Fatal(http.ListenAndServe(":"+core.Port, mux))
}
