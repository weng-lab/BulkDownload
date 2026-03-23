# Slice 3: Remove generic job mutation and finish cleanup

## Dependencies

Slice 2: Move tarball and script flows onto the lifecycle API

## Description

Delete `Jobs.Update`, trim any now-redundant manager mutation wrappers, and verify that the system still behaves the same through manager, API, and end-to-end coverage.

## Expected Behaviors Addressed

- A new job is still assembled by `Manager` with its ID, type, expiry, and file list before being stored.
- Callers can only perform the supported status and progress transitions, not arbitrary in-place job edits.
- Existing status polling and download behavior remain unchanged from a user perspective.

## Acceptance Criteria

- [x] `Jobs.Update` is removed.
- [x] Remaining manager helpers are either deleted or kept only where they improve readability over direct store calls.
- [x] Full test coverage for create, status, and download flows still passes.

## QA

1. Run the full `core` package tests.
2. Run API and end-to-end tests for create job, poll status, and download completed output.
3. Confirm there is no remaining compile-time or runtime dependency on generic job mutation.

---

*Appended after execution.*

## Completion

**Built:** Removed the public `Jobs.Update` entrypoint, deleted the now-unused manager mutation helpers, and trimmed the temporary store tests so the jobs store now exposes only the named lifecycle transitions.

**Decisions:** Kept the private `Jobs.update` helper to preserve shared locking for the explicit lifecycle methods while removing the generic mutation surface; used `go test ./...` to cover the core, API, and end-to-end flows called out in the slice QA.

**Deviations:** None.

**Files:** Modified `core/jobs.go`, `core/manager.go`, `core/jobs_test.go`, and `.plans/jobs-api/completed/03-remove-generic-job-mutation-and-finish-cleanup.md`.

**Notes for next slice:** No pending slices remain in `.plans/jobs-api`; the lifecycle API cleanup is complete and the test suite passes with no remaining `Jobs.Update` references.
