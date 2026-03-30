# Slice 3: Hardening And Deterministic Tests

## Dependencies

Slice 2: Shutdown Cancels Running Jobs And Cleans Partial Outputs.

## Description

Strengthen the implementation with deterministic coverage around queueing, cancellation, and shutdown coordination so the new behavior is easy to trust and maintain. The focus is on proving the service transitions jobs correctly and cleans up artifacts reliably without timing-sensitive flakiness.

## Expected Behaviors Addressed

- When many jobs are submitted quickly, all of them are accepted and stored, but only 4 run at once.
- Extra jobs remain `pending` until one of the 4 running jobs finishes and frees a slot.
- On service shutdown, active jobs stop as soon as they hit the next cancellation check.
- If shutdown interrupts a job, its half-written artifact is deleted and the job does not appear as a completed download.

## Acceptance Criteria

- [ ] Automated tests cover the 4-job execution limit in a deterministic way.
- [ ] Automated tests cover cancellation cleanup for interrupted archive or script output.
- [ ] Automated tests cover manager shutdown waiting for launched job goroutines to exit.

## QA

1. Run the targeted service and artifact test suites for the new queue and shutdown behavior.
2. Confirm the tests pass consistently on repeated runs.
3. Review the added tests to ensure they verify observable job state transitions and file cleanup, not only internal implementation details.

---

*Appended after execution.*

## Completion

**Built:** Added deterministic low-level cancellation tests for the copy loop so queue and shutdown behavior now has direct coverage at the artifact layer, in addition to the existing service-level queue and shutdown tests.

**Decisions:** Focused this slice on the most timing-sensitive part of the implementation: cancellation during buffered copy. Used controlled test doubles for the reader and writer so cancellation can be triggered at exact points without relying on slow files or sleeps.

**Deviations:** Did not add more service-layer timing tests because the current queue and shutdown coverage was already good after slices 1 and 2. The biggest remaining risk was low-level copy cancellation, so this slice tightened that area directly.

**Files:** Modified `internal/artifacts/progress_test.go`.

**Notes for next slice:** The artifact layer now has explicit cancellation coverage before read and before write. Slice 4 can focus on service cleanup/refactoring without needing to expand queue/shutdown behavior tests further.
