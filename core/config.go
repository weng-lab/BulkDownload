package core

import (
	"os"
	"time"
)

type Config struct {
	JobsDir         string
	SourceRootDir   string
	PublicBaseURL   string
	DownloadRootDir string
	Port            string
	JobTTL          time.Duration
	CleanupTick     time.Duration
}

func defaultConfig() Config {
	return Config{
		JobsDir:         "./jobs",
		SourceRootDir:   "",
		PublicBaseURL:   "https://download.mohd.org",
		DownloadRootDir: "mohd_data",
		Port:            "8080",
		JobTTL:          24 * time.Hour,
		CleanupTick:     5 * time.Minute,
	}
}

func LoadConfig() (Config, error) {
	config := defaultConfig()

	config.JobsDir = loadStringEnv("JOBS_DIR", config.JobsDir)
	config.SourceRootDir = loadStringEnv("SOURCE_ROOT_DIR", config.SourceRootDir)
	config.PublicBaseURL = loadStringEnv("PUBLIC_BASE_URL", config.PublicBaseURL)
	config.DownloadRootDir = loadStringEnv("DOWNLOAD_ROOT_DIR", config.DownloadRootDir)
	config.Port = loadStringEnv("PORT", config.Port)

	jobTTL, err := loadDurationEnv("JOB_TTL", config.JobTTL)
	if err != nil {
		return Config{}, err
	}
	config.JobTTL = jobTTL

	cleanupTick, err := loadDurationEnv("CLEANUP_TICK", config.CleanupTick)
	if err != nil {
		return Config{}, err
	}
	config.CleanupTick = cleanupTick

	return config, nil
}

func loadStringEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func loadDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}

	return duration, nil
}
