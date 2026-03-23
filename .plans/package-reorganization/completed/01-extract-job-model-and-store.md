# Slice 1: Extract Job Model And Store

## Dependencies

None.

## Description

Move the job model, job type and status definitions, job errors, in-memory store, and expiration logic into `internal/jobs`, then rewire the app so the existing routes and tests continue to work without behavioral changes.

## Expected Behaviors Addressed

- A developer can quickly find job state and storage logic in `internal/jobs`.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Acceptance Criteria

- [x] Job types, statuses, errors, store behavior, and expiration queries live in `internal/jobs`.
- [x] `main.go`, `api`, and relevant tests use `internal/jobs` instead of `core` for job state and store access.
- [x] Existing job lifecycle behavior remains unchanged.

## QA

1. Run the job and manager related tests and confirm they pass.
2. Run the API and end-to-end tests that cover job creation, status, and downloads.
3. Confirm production code no longer imports `core` for job types or the in-memory job store.

---

*Appended after execution.*

## Completion

**Built:** Moved the job model, statuses, errors, in-memory store, and expiration queries into `internal/jobs`, then rewired the app and tests to use the new package without changing runtime behavior.

**Decisions:** Kept `core.Manager` and cleanup orchestration in place for this slice, updated production callers to import `internal/jobs` directly, and added a test-only alias shim in `core` so existing manager/archive/script tests could stay focused on orchestration instead of package plumbing.

**Deviations:** Added `core/jobs_alias_test.go` as a temporary test-only compatibility layer so the refactor could stay scoped to job/store extraction; production code no longer depends on `core` for job state.

**Files:** Created `internal/jobs/jobs.go`, `internal/jobs/jobs_test.go`, and `core/jobs_alias_test.go`; removed `core/jobs.go` and `core/jobs_test.go`; updated `main.go`, `api/handlers.go`, `api/types.go`, `api/handlers_test.go`, `e2e_test.go`, `core/manager.go`, `core/cleanup.go`, `core/archive.go`, `core/script.go`, and `core/helpers_test.go`.

**Notes for next slice:** `core` still owns config loading plus service/artifact orchestration, but job state now sits behind `internal/jobs`; later slices can keep importing `internal/jobs` directly and remove the test alias shim once more `core` tests move or are rewritten.
