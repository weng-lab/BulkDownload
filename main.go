package main

import (
	"log"
	"net/http"
	"os"
)

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
