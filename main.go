package main

import (
	"context"
	"errors"
	"log"
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
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		log.Fatalf("create jobs dir: %v", err)
	}

	jobStore := jobs.NewJobs()
	manager := service.NewManager(jobStore, config)
	stopCleanup := service.StartCleanup(jobStore, config.JobsDir, config.CleanupTick)

	server := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           api.NewRouter(manager, jobStore, config),
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
		ReadTimeout:       httpServerReadTimeout,
		IdleTimeout:       httpServerIdleTimeout,
	}

	log.Printf(
		"config: jobs_dir=%s source_root_dir=%s public_base_url=%s download_root_dir=%s port=%s job_ttl=%s cleanup_tick=%s\n",
		config.JobsDir,
		config.SourceRootDir,
		config.PublicBaseURL,
		config.DownloadRootDir,
		config.Port,
		config.JobTTL,
		config.CleanupTick,
	)
	log.Printf("bulk download service listening on :%s", config.Port)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrCh:
		stopCleanup()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
		return
	case <-ctx.Done():
		stopCleanup()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpServerShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shutdown server: %v", err)
		}

		if err := <-serverErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
