package core

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var (
	JobsDir         = "./jobs"
	PublicBaseURL   = "https://download.mohd.org"
	DownloadRootDir = "mohd_data"
	Port            = "8080"
	ZipTTL          = 24 * time.Hour
	CleanupTick     = 5 * time.Minute
	// ProcessingDelay is mainly useful in tests to simulate zip work taking time.
	ProcessingDelay time.Duration
)

func LoadConfig() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("config: failed to load .env: %v", err)
	}

	JobsDir = loadStringEnv("JOBS_DIR", "./jobs")
	PublicBaseURL = loadStringEnv("PUBLIC_BASE_URL", "https://download.mohd.org")
	DownloadRootDir = loadStringEnv("DOWNLOAD_ROOT_DIR", "mohd_data")
	Port = loadStringEnv("PORT", "8080")
	ZipTTL = loadDurationEnv("ZIP_TTL", 24*time.Hour)
	CleanupTick = loadDurationEnv("CLEANUP_TICK", 5*time.Minute)
	ProcessingDelay = loadDurationEnv("PROCESSING_DELAY", 0)
}

func loadStringEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func loadDurationEnv(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("config: invalid %s=%q, using %s", key, value, fallback)
		return fallback
	}

	return duration
}
