package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestNewHTTPServerUsesSafeTimeoutDefaults(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	handler := http.NewServeMux()
	server := newHTTPServer(config, handler)

	if got, want := server.Addr, ":"+config.Port; got != want {
		t.Fatalf("server addr = %q, want %q", got, want)
	}
	if server.Handler != handler {
		t.Fatal("server handler did not match input handler")
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
}

type fakeLifecycleServer struct {
	listenStarted chan struct{}
	listenErrCh   chan error

	mu              sync.Mutex
	shutdownCalled  bool
	shutdownCtx     context.Context
	shutdownErr     error
	shutdownRelease chan struct{}
}

func newFakeLifecycleServer() *fakeLifecycleServer {
	return &fakeLifecycleServer{
		listenStarted:   make(chan struct{}),
		listenErrCh:     make(chan error, 1),
		shutdownRelease: make(chan struct{}),
	}
}

func (s *fakeLifecycleServer) ListenAndServe() error {
	close(s.listenStarted)
	return <-s.listenErrCh
}

func (s *fakeLifecycleServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.shutdownCalled = true
	s.shutdownCtx = ctx
	shutdownErr := s.shutdownErr
	release := s.shutdownRelease
	s.mu.Unlock()

	if shutdownErr != nil {
		return shutdownErr
	}

	if release != nil {
		<-release
	}

	s.listenErrCh <- http.ErrServerClosed
	return nil
}

func (s *fakeLifecycleServer) shutdownState() (bool, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdownCalled, s.shutdownCtx
}

func TestServeUntilShutdown_ShutsDownGracefullyOnContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newFakeLifecycleServer()
	cleanupStopped := make(chan struct{}, 1)
	stopCleanup := func() {
		cleanupStopped <- struct{}{}
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveUntilShutdown(ctx, server, stopCleanup)
	}()

	select {
	case <-server.listenStarted:
	case <-time.After(time.Second):
		t.Fatal("ListenAndServe was not called")
	}

	cancel()
	close(server.shutdownRelease)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serveUntilShutdown() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serveUntilShutdown did not return after cancellation")
	}

	select {
	case <-cleanupStopped:
	default:
		t.Fatal("stopCleanup was not called during shutdown")
	}

	shutdownCalled, shutdownCtx := server.shutdownState()
	if !shutdownCalled {
		t.Fatal("Shutdown was not called")
	}
	deadline, ok := shutdownCtx.Deadline()
	if !ok {
		t.Fatal("Shutdown context did not have a deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > httpServerShutdownTimeout {
		t.Fatalf("Shutdown deadline remaining = %s, want between 0 and %s", remaining, httpServerShutdownTimeout)
	}
}

func TestServeUntilShutdown_ReturnsListenErrorAndStopsCleanup(t *testing.T) {
	t.Parallel()

	server := newFakeLifecycleServer()
	wantErr := errors.New("listen failed")
	server.listenErrCh <- wantErr

	cleanupCalls := 0
	stopCleanup := func() {
		cleanupCalls++
	}

	err := serveUntilShutdown(context.Background(), server, stopCleanup)
	if !errors.Is(err, wantErr) {
		t.Fatalf("serveUntilShutdown() error = %v, want %v", err, wantErr)
	}
	if cleanupCalls != 1 {
		t.Fatalf("stopCleanup calls = %d, want 1", cleanupCalls)
	}
	if shutdownCalled, _ := server.shutdownState(); shutdownCalled {
		t.Fatal("Shutdown was called for listen failure")
	}
}

func TestServeUntilShutdown_PropagatesShutdownError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newFakeLifecycleServer()
	wantErr := errors.New("shutdown failed")
	server.shutdownErr = wantErr

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveUntilShutdown(ctx, server, nil)
	}()

	select {
	case <-server.listenStarted:
	case <-time.After(time.Second):
		t.Fatal("ListenAndServe was not called")
	}

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, wantErr) {
			t.Fatalf("serveUntilShutdown() error = %v, want wrapped %v", err, wantErr)
		}
	case <-time.After(time.Second):
		t.Fatal("serveUntilShutdown did not return after shutdown failure")
	}
}

func TestNewShutdownContextRegistersExpectedSignals(t *testing.T) {
	t.Parallel()

	if len(shutdownSignals) != 2 {
		t.Fatalf("len(shutdownSignals) = %d, want 2", len(shutdownSignals))
	}
	if shutdownSignals[0] != os.Interrupt {
		t.Fatalf("shutdownSignals[0] = %v, want %v", shutdownSignals[0], os.Interrupt)
	}
	if shutdownSignals[1] != os.Signal(syscall.SIGTERM) {
		t.Fatalf("shutdownSignals[1] = %v, want %v", shutdownSignals[1], syscall.SIGTERM)
	}

	ctx, cancel := newShutdownContext(context.Background())
	defer cancel()

	if ctx == nil {
		t.Fatal("newShutdownContext returned nil context")
	}
}
