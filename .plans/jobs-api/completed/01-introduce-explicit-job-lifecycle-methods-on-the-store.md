# Slice 1: Introduce explicit job lifecycle methods on the store

## Dependencies

None

## Description

Add named lifecycle operations to `Jobs`, prove them with focused store tests, and migrate the zip execution path to use them while keeping the old generic mutation path available for the remaining job types.

## Expected Behaviors Addressed

- Job execution paths update state through explicit store lifecycle methods instead of a generic mutation callback.
- Callers can only perform the supported status/progress transitions, not arbitrary in-place job edits.

## Acceptance Criteria

- [x] `Jobs` exposes `MarkProcessing`, `SetProgress`, `MarkFailed`, and `MarkDone`.
- [x] Store tests cover those methods, including progress clamping and completion and failure field updates.
- [x] Zip job execution uses the new lifecycle API and still completes successfully.
- [x] `Jobs.Update` remains temporarily so tarball and script paths still build.

## QA

1. Run the store tests for `core/jobs.go` and confirm the new lifecycle methods are covered.
2. Run the zip-focused manager and archive tests.
3. Create a zip job through the existing flow and confirm it still progresses from pending to processing to done.

---

*Appended after execution.*

## Completion

**Built:** Added explicit `Jobs` lifecycle methods for processing, progress, failure, and completion; covered them with focused store tests; and moved the zip execution path onto the new store API while leaving `Jobs.Update` in place for tarball and script flows.

**Decisions:** Kept a private `Jobs.update` helper so the new named lifecycle methods and the temporary `Jobs.Update` path share the same locking behavior; only the zip flow was migrated in this slice so the remaining job types still exercise the legacy mutation API as planned.

**Deviations:** None.

**Files:** Modified `core/jobs.go`, `core/archive.go`, `core/jobs_test.go`, and `.plans/jobs-api/completed/01-introduce-explicit-job-lifecycle-methods-on-the-store.md`.

**Notes for next slice:** Tarball and script execution still go through the manager mutation wrappers backed by `Jobs.Update`; slice 2 can migrate those paths directly to the new lifecycle methods and then verify no production flow still depends on the generic callback API.
