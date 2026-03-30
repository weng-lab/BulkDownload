# Slice 3: Job create flow

## Dependencies

Slice 2: Request backbone.

## Description

Use the request-scoped logger in the create, status, and download handlers to log only the important high-level branches. This makes each endpoint easy to follow in stdout without adding noisy helper-level logs.

## Expected Behaviors Addressed

- Each endpoint logs only the major steps and outcomes that matter, such as job accepted, job not found, job not ready, or download served.

## Acceptance Criteria

- [x] The create handler logs high-level outcomes such as accepted job and failed dispatch.
- [x] The status and download handlers log meaningful branches such as not found, not ready, and served download.
- [x] Endpoint logs use human-readable messages plus structured fields like `job_id` and `job_type` where relevant.

## QA

1. Start the service.
2. Create a valid job and confirm the handler logs that the job was accepted.
3. Request status for a missing job and confirm the handler logs that the job was not found.
4. Request a download before a job is ready and confirm the handler logs that the job is not ready.
5. Request a finished download and confirm the handler logs that the download was served.

---

*Appended after execution.*

## Completion

**Built:** Moved the create, status, and download endpoint logs onto the request-scoped `slog` logger. The handlers now emit human-readable high-level logs with request fields plus job-specific fields where relevant.

**Decisions:** Kept the change explicit in the handlers rather than adding a logging abstraction. Added one tiny `requestLogger` helper in `api/request_logging.go` so each handler can fetch the context logger with a default fallback.

**Deviations:** Did not add new logging-specific tests. I kept the existing handler behavior tests and verified the new logging shape manually by running the service and checking the emitted lines.

**Files:** Modified `api/handlers.go`, `api/handlers_test.go`, and `api/request_logging.go`.

**Notes for next slice:** Endpoint logs now consistently use `slog`, but async job logs in `internal/service/manager.go` still use stdlib `log.Printf`. Slice 4 can migrate those to the shared logger and keep the same field style with `job_id` and `job_type`.
