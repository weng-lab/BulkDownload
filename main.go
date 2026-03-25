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

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func newHTTPServer(config appconfig.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + config.Port,
		Handler:           handler,
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
		ReadTimeout:       httpServerReadTimeout,
		IdleTimeout:       httpServerIdleTimeout,
	}
}

type lifecycleServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func newShutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, shutdownSignals...)
}

func serveUntilShutdown(ctx context.Context, server lifecycleServer, stopCleanup func()) error {
	var stopOnce sync.Once
	safeStopCleanup := func() {
		if stopCleanup == nil {
			return
		}
		stopOnce.Do(stopCleanup)
	}
	defer safeStopCleanup()

	serverErrCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
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

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		return <-serverErrCh
	}
}

func run(ctx context.Context, config appconfig.Config) error {
	if err := os.MkdirAll(config.JobsDir, 0o755); err != nil {
		return fmt.Errorf("create jobs dir: %w", err)
	}

	jobStore := jobs.NewJobs()
	manager := service.NewManager(jobStore, config)
	stopCleanup := service.StartCleanup(jobStore, config.JobsDir, config.CleanupTick)
	server := newHTTPServer(config, api.NewRouter(manager, jobStore, config))

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

	return serveUntilShutdown(ctx, server, stopCleanup)
}

func main() {
	config, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, stop := newShutdownContext(context.Background())
	defer stop()

	if err := run(ctx, config); err != nil {
		log.Fatal(err)
	}
}
