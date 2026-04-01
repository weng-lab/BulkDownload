package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jair/bulkdownload/api"
	appconfig "github.com/jair/bulkdownload/internal/config"
	"github.com/jair/bulkdownload/internal/jobs"
	"github.com/jair/bulkdownload/internal/service"
)

const (
	httpServerReadHeaderTimeout = 5 * time.Second
	httpServerReadTimeout       = 30 * time.Second
	httpServerIdleTimeout       = 2 * time.Minute
	httpServerShutdownTimeout   = 10 * time.Second
)

func main() {
	config, err := appconfig.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := newLogger(os.Stdout, config.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create logger: %v\n", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		logger.Error("create jobs directory failed", "error", err, "jobs_dir", config.JobsDir)
		os.Exit(1)
	}

	jobStore := jobs.NewJobs()
	manager := service.NewManager(jobStore, config)
	stopCleanup := service.StartCleanup(jobStore, config.JobsDir, config.CleanupTick)

	server := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           api.NewRouter(logger, manager, jobStore, config),
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
		ReadTimeout:       httpServerReadTimeout,
		IdleTimeout:       httpServerIdleTimeout,
	}

	logger.Info(
		"service configuration loaded",
		"jobs_dir", config.JobsDir,
		"source_root_dir", config.SourceRootDir,
		"public_base_url", config.PublicBaseURL,
		"download_root_dir", config.DownloadRootDir,
		"port", config.Port,
		"log_level", config.LogLevel,
		"job_ttl", config.JobTTL,
		"cleanup_tick", config.CleanupTick,
	)
	logger.Info("service listening", "addr", ":"+config.Port)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrCh:
		stopCleanup()
		manager.Shutdown()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server exited unexpectedly", "error", err)
			os.Exit(1)
		}
		logger.Info("server stopped")
		return
	case <-ctx.Done():
		stopCleanup()
		logger.Info("shutdown started")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpServerShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown server failed", "error", err)
			os.Exit(1)
		}

		if err := <-serverErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped with error", "error", err)
			os.Exit(1)
		}

		manager.Shutdown()

		logger.Info("shutdown complete")
	}
}
