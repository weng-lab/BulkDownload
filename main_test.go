package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/jair/bulkdownload/internal/jobs"
)

func TestRun_ShutsDownGracefullyOnContextCancellation(t *testing.T) {
	originalServeHTTPServer := serveHTTPServer
	originalShutdownHTTPServer := shutdownHTTPServer
	originalStartCleanup := startCleanup
	t.Cleanup(func() {
		serveHTTPServer = originalServeHTTPServer
		shutdownHTTPServer = originalShutdownHTTPServer
		startCleanup = originalStartCleanup
	})

	config := testConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveStarted := make(chan *http.Server, 1)
	serveReleased := make(chan struct{})
	serveHTTPServer = func(server *http.Server) error {
		serveStarted <- server
		<-serveReleased
		return http.ErrServerClosed
	}

	cleanupStopped := make(chan struct{}, 1)
	startCleanup = func(_ *jobs.Jobs, _ string, _ time.Duration) func() {
		return func() {
			cleanupStopped <- struct{}{}
		}
	}

	shutdownCtxCh := make(chan context.Context, 1)
	shutdownHTTPServer = func(_ *http.Server, shutdownCtx context.Context) error {
		shutdownCtxCh <- shutdownCtx
		close(serveReleased)
		return nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, config)
	}()

	var server *http.Server
	select {
	case server = <-serveStarted:
	case <-time.After(time.Second):
		t.Fatal("run did not start serving")
	}

	if got, want := server.Addr, ":"+config.Port; got != want {
		t.Fatalf("server addr = %q, want %q", got, want)
	}
	if server.Handler == nil {
		t.Fatal("server handler was nil")
	}
	if got := server.ReadHeaderTimeout; got != httpServerReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", got, httpServerReadHeaderTimeout)
	}
	if got := server.ReadTimeout; got != httpServerReadTimeout {
		t.Fatalf("ReadTimeout = %s, want %s", got, httpServerReadTimeout)
	}
	if got := server.IdleTimeout; got != httpServerIdleTimeout {
		t.Fatalf("IdleTimeout = %s, want %s", got, httpServerIdleTimeout)
	}
	if got := server.WriteTimeout; got != 0 {
		t.Fatalf("WriteTimeout = %s, want 0", got)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not return after cancellation")
	}

	select {
	case <-cleanupStopped:
	default:
		t.Fatal("stopCleanup was not called during shutdown")
	}

	select {
	case shutdownCtx := <-shutdownCtxCh:
		deadline, ok := shutdownCtx.Deadline()
		if !ok {
			t.Fatal("shutdown context did not have a deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > httpServerShutdownTimeout {
			t.Fatalf("shutdown deadline remaining = %s, want between 0 and %s", remaining, httpServerShutdownTimeout)
		}
	default:
		t.Fatal("shutdown was not called")
	}
}

func TestRun_ReturnsListenErrorAndStopsCleanup(t *testing.T) {
	originalServeHTTPServer := serveHTTPServer
	originalShutdownHTTPServer := shutdownHTTPServer
	originalStartCleanup := startCleanup
	t.Cleanup(func() {
		serveHTTPServer = originalServeHTTPServer
		shutdownHTTPServer = originalShutdownHTTPServer
		startCleanup = originalStartCleanup
	})

	wantErr := errors.New("listen failed")
	serveHTTPServer = func(_ *http.Server) error {
		return wantErr
	}

	cleanupCalls := 0
	startCleanup = func(_ *jobs.Jobs, _ string, _ time.Duration) func() {
		return func() {
			cleanupCalls++
		}
	}

	shutdownCalled := false
	shutdownHTTPServer = func(_ *http.Server, _ context.Context) error {
		shutdownCalled = true
		return nil
	}

	err := run(context.Background(), testConfig(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want %v", err, wantErr)
	}
	if cleanupCalls != 1 {
		t.Fatalf("stopCleanup calls = %d, want 1", cleanupCalls)
	}
	if shutdownCalled {
		t.Fatal("shutdown was called for listen failure")
	}
}

func TestRun_PropagatesShutdownError(t *testing.T) {
	originalServeHTTPServer := serveHTTPServer
	originalShutdownHTTPServer := shutdownHTTPServer
	originalStartCleanup := startCleanup
	t.Cleanup(func() {
		serveHTTPServer = originalServeHTTPServer
		shutdownHTTPServer = originalShutdownHTTPServer
		startCleanup = originalStartCleanup
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveStarted := make(chan struct{}, 1)
	serveReleased := make(chan struct{})
	serveHTTPServer = func(_ *http.Server) error {
		serveStarted <- struct{}{}
		<-serveReleased
		return http.ErrServerClosed
	}

	cleanupStopped := make(chan struct{}, 1)
	startCleanup = func(_ *jobs.Jobs, _ string, _ time.Duration) func() {
		return func() {
			cleanupStopped <- struct{}{}
		}
	}

	wantErr := errors.New("shutdown failed")
	shutdownHTTPServer = func(_ *http.Server, _ context.Context) error {
		close(serveReleased)
		return wantErr
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, testConfig(t))
	}()

	select {
	case <-serveStarted:
	case <-time.After(time.Second):
		t.Fatal("run did not start serving")
	}

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, wantErr) {
			t.Fatalf("run() error = %v, want wrapped %v", err, wantErr)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not return after shutdown failure")
	}

	select {
	case <-cleanupStopped:
	default:
		t.Fatal("stopCleanup was not called before shutdown")
	}
}

func TestRun_ServesRequestsAndShutsDownGracefully(t *testing.T) {
	config := testConfig(t)
	config.Port = freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, config)
	}()

	statusURL := "http://127.0.0.1:" + config.Port + "/status/missing"
	deadline := time.Now().Add(2 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		resp, err := http.Get(statusURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusNotFound)
			}
			ready = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		t.Fatal("server did not start listening in time")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not shut down cleanly")
	}
}

func TestShutdownSignalsIncludeInterruptAndSigterm(t *testing.T) {
	if len(shutdownSignals) != 2 {
		t.Fatalf("len(shutdownSignals) = %d, want 2", len(shutdownSignals))
	}
	if shutdownSignals[0] != os.Interrupt {
		t.Fatalf("shutdownSignals[0] = %v, want %v", shutdownSignals[0], os.Interrupt)
	}
	if shutdownSignals[1] != os.Signal(syscall.SIGTERM) {
		t.Fatalf("shutdownSignals[1] = %v, want %v", shutdownSignals[1], syscall.SIGTERM)
	}
}

func freePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", listener.Addr())
	}

	return strconv.Itoa(addr.Port)
}
