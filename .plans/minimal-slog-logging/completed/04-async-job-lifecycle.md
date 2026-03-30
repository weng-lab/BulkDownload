# Slice 4: Async job lifecycle

## Dependencies

Slice 1: Tracer logger.

## Description

Log async job execution through the shared logger with job-scoped fields so background work remains readable and easy to correlate. Keep the logging limited to major lifecycle events: job started, job completed, and job failed.

## Expected Behaviors Addressed

- Each async job logs its own lifecycle independently, including start, completion, and failure, tied together by `job_id`.

## Acceptance Criteria

- [x] Zip, tarball, and script jobs log a clear start event.
- [x] Successful jobs log a clear completion event.
- [x] Failed jobs log a clear error event with `job_id` and `job_type`.

## QA

1. Start the service.
2. Create one successful job and follow the logs until completion.
3. Trigger one failing job path if practical and follow the logs until failure.
4. Confirm the async logs are easy to correlate by `job_id` and show only the main lifecycle steps.

---

*Appended after execution.*

## Completion

**Built:** Moved async zip, tarball, and script lifecycle logging onto the shared `slog` path. Background jobs now log `started`, `completed`, and `failed` with `job_id` and `job_type`.

**Decisions:** Kept the logging in the dispatch goroutines instead of spreading it through the artifact execution helpers. Set the shared app logger as the default logger in `main` so background work can use `slog.Default()` without threading a logger through more constructors.

**Deviations:** Did not add logging-specific tests. Verified the lifecycle logs manually with one successful zip job and one failing script job by making the jobs directory read-only after startup.

**Files:** Modified `main.go` and `internal/service/manager.go`.

**Notes for next slice:** Cleanup still has no `slog` visibility. Slice 5 can use the same `slog.Default()` approach for debug sweep logs, info removal logs, and cleanup errors.
