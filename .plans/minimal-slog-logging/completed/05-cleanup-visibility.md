# Slice 5: Cleanup visibility

## Dependencies

Slice 1: Tracer logger.

## Description

Add simple logging to the cleanup loop so periodic sweeps are visible at `DEBUG`, actual expired-job removals are visible at `INFO`, and cleanup failures are visible at `ERROR`. This gives operational visibility without adding noise to normal `INFO` output.

## Expected Behaviors Addressed

- Cleanup emits a debug log every sweep, an info log when it removes expired jobs, and error logs when cleanup work fails.

## Acceptance Criteria

- [x] Every cleanup sweep emits a `DEBUG` log.
- [x] Cleanup emits an `INFO` log when expired jobs are removed.
- [x] Cleanup emits an `ERROR` log when file cleanup or related work fails.

## QA

1. Start the service with `LOG_LEVEL=debug`.
2. Wait for at least one cleanup interval and confirm a sweep log is emitted.
3. Create or prepare an expired job and confirm cleanup logs an info-level removal event.
4. Trigger a cleanup failure path if practical and confirm an error log is emitted.

---

*Appended after execution.*

## Completion

**Built:** Added cleanup loop logging with a debug log at the start of each sweep, info logs when expired jobs are removed, and error logs when artifact deletion fails.

**Decisions:** Kept the implementation in `cleanup.go` and used `slog.Default()` directly to avoid threading a logger through more call sites. Logged both per-job removals and an aggregate removed-count line so cleanup activity is visible without adding extra abstractions.

**Deviations:** I verified the debug and info paths manually, but did not trigger a real cleanup deletion error during QA. The error path is implemented directly on `artifacts.CleanupFile` failure and covered by code inspection rather than a dedicated log-output test.

**Files:** Modified `internal/service/cleanup.go`.

**Notes for next slice:** All planned logging slices are now complete. Remaining work, if any, is polish or follow-up tweaks rather than part of the original slice plan.
