package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	ErrInvalidJobTTL      = errors.New("invalid JOB_TTL")
	ErrInvalidCleanupTick = errors.New("invalid CLEANUP_TICK")
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
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	config := defaultConfig()

	config.JobsDir = loadStringEnv("JOBS_DIR", config.JobsDir)
	config.SourceRootDir = loadStringEnv("SOURCE_ROOT_DIR", config.SourceRootDir)
	config.PublicBaseURL = loadStringEnv("PUBLIC_BASE_URL", config.PublicBaseURL)
	config.DownloadRootDir = loadStringEnv("DOWNLOAD_ROOT_DIR", config.DownloadRootDir)
	config.Port = loadStringEnv("PORT", config.Port)

	jobTTL, err := loadDurationEnv(config.JobTTL, "JOB_TTL")
	if err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalidJobTTL, err)
	}
	config.JobTTL = jobTTL

	cleanupTick, err := loadDurationEnv(config.CleanupTick, "CLEANUP_TICK")
	if err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalidCleanupTick, err)
	}
	config.CleanupTick = cleanupTick

	return config, nil
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("parse %s: invalid line %q", path, line)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("parse %s: empty key in line %q", path, line)
		}
		if current, exists := os.LookupEnv(key); exists && current != "" {
			continue
		}

		value = strings.TrimSpace(value)
		value = trimMatchingQuotes(value)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s: %w", key, path, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", path, err)
	}

	return nil
}

func loadStringEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func loadDurationEnv(fallback time.Duration, keys ...string) (time.Duration, error) {
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if !ok || value == "" {
			continue
		}

		duration, err := time.ParseDuration(value)
		if err != nil {
			return 0, err
		}

		return duration, nil
	}

	return fallback, nil
}

func trimMatchingQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return strings.Trim(value, `"`)
	}
	if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
		return strings.Trim(value, `'`)
	}
	return value
}
