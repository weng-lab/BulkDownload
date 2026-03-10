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

	if err := os.MkdirAll(core.OutputDir, 0o755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	store := core.NewStore()
	core.StartCleanup(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/zip", api.HandleCreateZip(store))
	mux.HandleFunc("/status/", api.HandleStatus(store))
	mux.HandleFunc("/download/", api.HandleDownload(store))

	port := os.Getenv("PORT")
	if port == "" {
		port = core.DefaultPort
	}

	log.Printf("config: zip_ttl=%s cleanup_tick=%s processing_delay=%s", core.ZipTTL, core.CleanupTick, core.ProcessingDelay)
	log.Printf("bulk download service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
