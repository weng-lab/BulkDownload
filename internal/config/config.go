package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	ErrInvalidJobTTL      = errors.New("invalid JOB_TTL")
	ErrInvalidCleanupTick = errors.New("invalid CLEANUP_TICK")
)

const (
	envJobsDir         = "JOBS_DIR"
	envSourceRootDir   = "SOURCE_ROOT_DIR"
	envPublicBaseURL   = "PUBLIC_BASE_URL"
	envDownloadRootDir = "DOWNLOAD_ROOT_DIR"
	envPort            = "PORT"
	envLogLevel        = "LOG_LEVEL"
	envJobTTL          = "JOB_TTL"
	envCleanupTick     = "CLEANUP_TICK"
)

type Config struct {
	JobsDir         string
	SourceRootDir   string
	PublicBaseURL   string
	DownloadRootDir string
	Port            string
	LogLevel        string
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
		LogLevel:        "info",
		JobTTL:          24 * time.Hour,
		CleanupTick:     5 * time.Minute,
	}
}

func LoadConfig() (Config, error) {
	dotEnv, err := godotenv.Read(".env")
	if err != nil {
		if os.IsNotExist(err) {
			dotEnv = map[string]string{}
		} else {
			return Config{}, fmt.Errorf("read .env: %w", err)
		}
	}

	return resolveConfig(mergeEnvWithOverride(dotEnv, currentEnv()))
}

func resolveConfig(env map[string]string) (Config, error) {
	config := defaultConfig()

	config.JobsDir = resolveString(env, envJobsDir, config.JobsDir)
	config.SourceRootDir = resolveString(env, envSourceRootDir, config.SourceRootDir)
	config.PublicBaseURL = resolveString(env, envPublicBaseURL, config.PublicBaseURL)
	config.DownloadRootDir = resolveString(env, envDownloadRootDir, config.DownloadRootDir)
	config.Port = resolveString(env, envPort, config.Port)
	config.LogLevel = resolveString(env, envLogLevel, config.LogLevel)

	jobTTL, err := resolveDuration(env, config.JobTTL, envJobTTL)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalidJobTTL, err)
	}
	config.JobTTL = jobTTL

	cleanupTick, err := resolveDuration(env, config.CleanupTick, envCleanupTick)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalidCleanupTick, err)
	}
	config.CleanupTick = cleanupTick

	return config, nil
}

func currentEnv() map[string]string {
	env := make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	return env
}

func mergeEnvWithOverride(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	maps.Copy(merged, base)
	maps.Copy(merged, override)
	return merged
}

func resolveString(env map[string]string, key, fallback string) string {
	value, ok := env[key]
	if !ok {
		return fallback
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func resolveDuration(env map[string]string, fallback time.Duration, key string) (time.Duration, error) {
	value, ok := env[key]
	if !ok {
		return fallback, nil
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	if duration <= 0 {
		return 0, fmt.Errorf("must be greater than 0")
	}

	return duration, nil
}
