# Slice 1: Explicit HTTP server with safe defaults

## Dependencies

None.

## Description

Replace the bare `ListenAndServe` startup path with an explicit `http.Server` that keeps the current routes and startup behavior intact while adding hardcoded timeout defaults. This delivers a safer HTTP entry point without changing config, job execution, or download behavior.

## Expected Behaviors Addressed

- The service starts with an explicit HTTP server instead of the bare `ListenAndServe` helper.
- The server applies protective read-side and idle timeouts by default.
- Large `/download` responses are not cut off by an aggressive write timeout.

## Acceptance Criteria

- [x] The main startup path constructs and runs an explicit `http.Server` instead of calling `http.ListenAndServe` directly.
- [x] The server uses hardcoded `ReadHeaderTimeout`, `ReadTimeout`, and `IdleTimeout` values.
- [x] The server leaves `WriteTimeout` unset or zero so completed downloads are not prematurely terminated.
- [x] Existing HTTP behavior still works end-to-end for create, status, and download flows.

## QA

1. Run the Go test suite.
2. Start the service locally.
3. Create a job with `POST /jobs` and confirm the response is still `202 Accepted`.
4. Poll `GET /status/{id}` until the job completes, then call `GET /download/{id}` and confirm the artifact still downloads successfully.
5. Review the startup path and confirm the server is created through `http.Server` with the intended timeout values.

## Completion

**Built:** Replaced the bare `http.ListenAndServe` startup call with an explicit `http.Server` configured with hardcoded read-side and idle timeout defaults while preserving the existing job routes.

**Decisions:** Extracted router and server construction into helpers so the startup wiring is testable, and left `WriteTimeout` unset to avoid interrupting large downloads.

**Deviations:** Did not run a separate manual curl session because the existing end-to-end Go tests already verify create, status, and download behavior for the supported job types.

**Files:** `main.go`, `main_test.go`

**Notes for next slice:** The startup path now owns an explicit `*http.Server` and cleanup stop function, so signal handling and coordinated shutdown can build on that wiring directly.
