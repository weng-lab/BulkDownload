# Slice 2: Shutdown Cancels Running Jobs And Cleans Partial Outputs

## Dependencies

Slice 1: Cap Active Jobs At Four.

## Description

Teach running jobs to stop when the service shuts down. When shutdown interrupts archive creation or script writing, the job should stop promptly, remove any partial output file, and end in a failed state rather than looking like a completed download.

## Expected Behaviors Addressed

- On service shutdown, active jobs stop as soon as they hit the next cancellation check.
- If shutdown interrupts a job, its half-written artifact is deleted and the job does not appear as a completed download.
- On normal job failure, partial output files are still removed.

## Acceptance Criteria

- [ ] Service shutdown signals in-flight jobs to stop.
- [ ] A job interrupted during shutdown ends as `failed` and does not retain a completed filename.
- [ ] Any partially written `.zip`, `.tar.gz`, or `.sh` file is removed during cancellation cleanup.

## QA

1. Start the service with a job large enough to take noticeable time.
2. Submit a zip, tarball, or script job and wait until it has started processing.
3. Trigger service shutdown while the job is still running.
4. After shutdown completes, inspect the jobs directory.
5. Confirm the partial output file is gone.
6. Confirm the interrupted job reports `failed` rather than `done`.

---

*Appended after execution.*

## Completion

**Built:** Added manager-level shutdown cancellation and waiting, threaded context through job execution, made archive and script creation context-aware, and cleaned up partial output files when cancellation interrupts running work.

**Decisions:** Kept pending jobs untouched if shutdown reaches them before they start. Used a manager-owned root context plus waitgroup so dispatched goroutines can either stop while queued or fail cleanly once processing has begun. Added context-aware artifact helpers while preserving the old exported helpers as `context.Background()` wrappers.

**Deviations:** The manager shutdown test covers queued goroutines exiting cleanly, while direct execution tests cover failed status and artifact cleanup for interrupted work. That split kept the shutdown test deterministic without adding test-only hooks to production code.

**Files:** Modified `main.go`, `internal/service/manager.go`, `internal/service/archive.go`, `internal/service/script.go`, `internal/service/manager_test.go`, `internal/service/archive_test.go`, `internal/service/script_test.go`, `internal/artifacts/archive.go`, `internal/artifacts/script.go`, and `internal/artifacts/progress.go`.

**Notes for next slice:** Cancellation now relies on `Manager.ctx` and the new context-aware artifact helpers. The next test-hardening slice can add more deterministic low-level cancellation coverage around copy loops if needed.
