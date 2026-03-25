package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
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

var (
	shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
	notifyContext   = signal.NotifyContext
	serveHTTPServer = func(server *http.Server) error {
		return server.ListenAndServe()
	}
	shutdownHTTPServer = func(server *http.Server, ctx context.Context) error {
		return server.Shutdown(ctx)
	}
	startCleanup = service.StartCleanup
)

func run(ctx context.Context, config appconfig.Config) error {
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		return fmt.Errorf("create jobs dir: %w", err)
	}

	jobStore := jobs.NewJobs()
	manager := service.NewManager(jobStore, config)
	stopCleanup := startCleanup(jobStore, config.JobsDir, config.CleanupTick)
	var stopOnce sync.Once
	safeStopCleanup := func() {
		if stopCleanup == nil {
			return
		}
		stopOnce.Do(stopCleanup)
	}
	defer safeStopCleanup()

	server := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           api.NewRouter(manager, jobStore, config),
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
		ReadTimeout:       httpServerReadTimeout,
		IdleTimeout:       httpServerIdleTimeout,
	}

	log.Printf(
		"config: jobs_dir=%s source_root_dir=%s public_base_url=%s download_root_dir=%s port=%s job_ttl=%s cleanup_tick=%s",
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
		err := serveHTTPServer(server)
		if errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- nil
			return
		}
		serverErrCh <- err
	}()

	select {
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
		safeStopCleanup()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpServerShutdownTimeout)
		defer cancel()

		if err := shutdownHTTPServer(server, shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		return <-serverErrCh
	}
}

func main() {
	config, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, stop := notifyContext(context.Background(), shutdownSignals...)
	defer stop()

	if err := run(ctx, config); err != nil {
		log.Fatal(err)
	}
}
