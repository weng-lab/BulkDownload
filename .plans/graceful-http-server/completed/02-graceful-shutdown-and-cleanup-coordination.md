# Slice 2: Graceful shutdown and cleanup coordination

## Dependencies

Slice 1: Explicit HTTP server with safe defaults.

## Description

Add signal-driven shutdown to the HTTP server so the service stops accepting new requests on process termination, gives in-flight HTTP requests a bounded window to finish, and stops the background cleanup goroutine as part of shutdown. This keeps the slice focused on HTTP lifecycle coordination without introducing job draining or cancellation.

## Expected Behaviors Addressed

- On `SIGINT` or `SIGTERM`, the service stops accepting new requests and begins graceful shutdown.
- In-flight HTTP requests get a bounded window to finish before shutdown returns.
- Background cleanup work started by the service is stopped as part of shutdown coordination.
- Treat `http.ErrServerClosed` as the expected shutdown path rather than a fatal startup error.

## Acceptance Criteria

- [ ] The server listens until the process receives `SIGINT` or `SIGTERM`.
- [ ] Shutdown uses a bounded context and calls `srv.Shutdown` instead of exiting abruptly.
- [ ] The normal `http.ErrServerClosed` path is handled as an expected shutdown condition, not a fatal startup failure.
- [ ] The cleanup stop function is invoked during shutdown so the cleanup goroutine does not outlive the server lifecycle.
- [ ] No manager wait group, job draining, job cancellation, or shutdown-time job deletion behavior is introduced in this slice.

## QA

1. Run the Go test suite.
2. Start the service locally.
3. Send a normal request such as `POST /jobs` and confirm the service behaves as before.
4. Send `SIGINT` or `SIGTERM` to the process and confirm the service exits cleanly without a fatal `http.ErrServerClosed` log.
5. Confirm the shutdown path stops background cleanup work as part of process teardown.

## Completion

**Built:** Added signal-driven server lifecycle handling that keeps the HTTP server running until shutdown, stops background cleanup as part of teardown, and uses a bounded graceful shutdown window.

**Decisions:** Extracted `run`, `serveUntilShutdown`, and `newShutdownContext` helpers so shutdown behavior is testable without driving `main` directly, and used a `sync.Once` wrapper so cleanup shutdown only runs once.

**Deviations:** Verified signal registration and shutdown coordination through unit tests with a fake lifecycle server instead of sending real OS signals to `main`, which keeps the lifecycle tests deterministic.

**Files:** `main.go`, `main_test.go`, `.plans/graceful-http-server/completed/02-graceful-shutdown-and-cleanup-coordination.md`

**Notes for next slice:** HTTP shutdown now cleanly stops new requests and the cleanup goroutine, but manager job workers are still intentionally left uncancelled and undrained for a later slice.
