# Slice 2: Request backbone

## Dependencies

Slice 1: Tracer logger.

## Description

Add one small HTTP middleware that derives a request-scoped logger, stores it in request context, and emits one consistent finished-request log line for every request. This creates a clean request logging backbone without changing endpoint behavior.

## Expected Behaviors Addressed

- Every HTTP request emits one consistent finished-request log line with status and duration.

## Acceptance Criteria

- [x] Every routed HTTP request gets a request-scoped logger in context.
- [x] Every request emits exactly one finished-request log with stable fields such as `method`, `path`, `status`, and `duration_ms`.
- [x] The middleware stays small and does not introduce a broad logging abstraction.

## QA

1. Start the service.
2. Send a successful request to `POST /jobs`.
3. Send a request to `GET /status/{id}` for an unknown job.
4. Confirm each request produces one `request finished` log line with the correct method, path, status, and duration.

---

*Appended after execution.*

## Completion

**Built:** Added a small request logging middleware that attaches a request-scoped `slog` logger to context and emits one `request finished` line per request. Wired the router and server setup to use the shared logger for all HTTP requests.

**Decisions:** Kept the implementation in the `api` package so handlers can use the same context-scoped logger in later slices without adding a separate logging package. Used a minimal status recorder to capture the final HTTP status while defaulting to `200 OK` when handlers do not call `WriteHeader`.

**Deviations:** Existing handler-level stdlib logs still appear alongside the new request-finished lines. That is expected for this slice; slice 3 will migrate endpoint flow logs onto the request logger.

**Files:** Added `api/request_logging.go` and `api/request_logging_test.go`. Modified `api/router.go`, `main.go`, and `e2e_test.go`.

**Notes for next slice:** Handlers in `api` can now read the request logger from context via `loggerFromContext`. The request-finished backbone is already in place, so slice 3 only needs to replace handler-local `log.Printf` calls with request-scoped `slog` calls.
