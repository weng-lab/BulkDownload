package core

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	OutputDir   = "./zips"
	DefaultPort = "8080"
)

var (
	ZipTTL          = 24 * time.Hour
	CleanupTick     = 5 * time.Minute
	ProcessingDelay time.Duration
)

func LoadConfig() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("config: failed to load .env: %v", err)
	}

	ZipTTL = loadDurationEnv("ZIP_TTL", 24*time.Hour)
	CleanupTick = loadDurationEnv("CLEANUP_TICK", 5*time.Minute)
	ProcessingDelay = loadDurationEnv("PROCESSING_DELAY", 0)
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
