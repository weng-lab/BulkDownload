# Slice 5: Cleanup visibility

## Dependencies

Slice 1: Tracer logger.

## Description

Add simple logging to the cleanup loop so periodic sweeps are visible at `DEBUG`, actual expired-job removals are visible at `INFO`, and cleanup failures are visible at `ERROR`. This gives operational visibility without adding noise to normal `INFO` output.

## Expected Behaviors Addressed

- Cleanup emits a debug log every sweep, an info log when it removes expired jobs, and error logs when cleanup work fails.

## Acceptance Criteria

- [ ] Every cleanup sweep emits a `DEBUG` log.
- [ ] Cleanup emits an `INFO` log when expired jobs are removed.
- [ ] Cleanup emits an `ERROR` log when file cleanup or related work fails.

## QA

1. Start the service with `LOG_LEVEL=debug`.
2. Wait for at least one cleanup interval and confirm a sweep log is emitted.
3. Create or prepare an expired job and confirm cleanup logs an info-level removal event.
4. Trigger a cleanup failure path if practical and confirm an error log is emitted.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
