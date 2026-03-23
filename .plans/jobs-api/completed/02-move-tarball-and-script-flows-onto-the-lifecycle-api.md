# Slice 2: Move tarball and script flows onto the lifecycle API

## Dependencies

Slice 1: Introduce explicit job lifecycle methods on the store

## Description

Migrate the remaining job execution paths from the generic update callback to the explicit lifecycle methods so all job types use the same store API, while keeping manager dispatch behavior unchanged.

## Expected Behaviors Addressed

- Creating a job still goes through the explicit manager dispatch methods for zip, tarball, and script jobs.
- Job execution paths update state through explicit store lifecycle methods instead of a generic mutation callback.
- Existing status polling and download behavior remain unchanged from a user perspective.

## Acceptance Criteria

- [x] Tarball and script execution paths use the new lifecycle methods.
- [x] Existing public manager dispatch methods remain as-is.
- [x] Archive and script success and failure tests pass unchanged.
- [x] No production path still needs `Jobs.Update`.

## QA

1. Run the tarball and script manager tests.
2. Run archive and script failure-path tests.
3. Create tarball and script jobs through the existing flow and confirm status and output behavior stay the same.

---

*Appended after execution.*

## Completion

**Built:** Moved the tarball and script execution paths onto `Jobs.MarkProcessing`, `Jobs.SetProgress`, `Jobs.MarkFailed`, and `Jobs.MarkDone` so all live job execution flows now use the explicit lifecycle API while keeping the public dispatch methods unchanged.

**Decisions:** Kept the now-unused private manager mutation helpers in place for this slice so the change stayed focused on migrating the active execution paths; verification included targeted core lifecycle tests plus tarball and script end-to-end coverage.

**Deviations:** None.

**Files:** Modified `core/archive.go`, `core/script.go`, and `.plans/jobs-api/completed/02-move-tarball-and-script-flows-onto-the-lifecycle-api.md`.

**Notes for next slice:** `Jobs.Update` is no longer used by the active tarball, script, or zip execution paths. Slice 3 can remove `Jobs.Update`, delete the unused private manager mutation helpers in `core/manager.go`, and then trim the temporary `Jobs.Update` tests.
