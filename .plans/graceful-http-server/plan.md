# Graceful HTTP Server Startup And Shutdown

## Problem

I want the service startup path to stop using the bare `ListenAndServe` call, because it runs without defensive server timeouts and gives me no graceful shutdown path. Right now the process can only exit abruptly, which is not a safe default for a public HTTP service.

## Solution

Replace the direct server start with an explicit `http.Server` configured in the main startup path. Use hardcoded timeout values for now, wire in signal handling for `SIGINT` and `SIGTERM`, and shut the server down gracefully with a bounded timeout.

Keep this slice focused on the HTTP lifecycle only. Do not add config/env knobs yet, and do not add job-draining or cancellation behavior yet.

## Expected Behavior

- The service starts with an explicit HTTP server instead of the bare `ListenAndServe` helper.
- The server applies protective read-side and idle timeouts by default.
- Large `/download` responses are not cut off by an aggressive write timeout.
- On `SIGINT` or `SIGTERM`, the service stops accepting new requests and begins graceful shutdown.
- In-flight HTTP requests get a bounded window to finish before shutdown returns.
- Background cleanup work started by the service is stopped as part of shutdown coordination.

## Implementation Decisions

- Use an explicit `http.Server` in the main startup path rather than expanding the change across the rest of the app.
- Hardcode sensible timeout values for now instead of adding new env/config fields in this slice.
- Include `ReadHeaderTimeout` in addition to `ReadTimeout` and `IdleTimeout`.
- Leave `WriteTimeout` unset or zero so archive/script downloads are not prematurely terminated.
- Treat `http.ErrServerClosed` as the expected shutdown path rather than a fatal startup error.
- Coordinate shutdown from OS signals and stop the cleanup goroutine during that sequence.
- Defer job cancellation, job draining, wait groups, and any artifact deletion policy to a later follow-up, since current job workers are not cancellation-aware.

## Testing Approach

- Add tests around the startup/shutdown path in a way that does not require sending real process signals to `main` directly.
- Verify the server is built with the intended timeout values.
- Verify graceful shutdown uses a bounded context and does not report the normal server-closed path as a fatal error.
- Verify cleanup shutdown is coordinated correctly so background cleanup does not outlive server shutdown.
- Run the existing Go test suite to catch regressions in API behavior and service lifecycle wiring.

## Out of Scope

- Adding timeout settings to `.env` or config parsing.
- Introducing manager wait groups or blocking shutdown on long-running jobs.
- Cancelling in-progress archive/script generation.
- Deleting pending or processing jobs on shutdown.
- Persisting in-memory job state across process restarts.
