# Slice 4: Extract Service And Cleanup, Remove Core

## Dependencies

- Slice 1: Extract Job Model And Store.
- Slice 2: Extract Runtime Config.
- Slice 3: Extract Artifact Builders.

## Description

Move manager orchestration and cleanup startup into `internal/service`, wire `api` and `main.go` to the final package layout, and remove the remaining `core` package without changing the service's public behavior.

## Expected Behaviors Addressed

- HTTP handlers remain focused on request and response flow.
- `main.go` still wires the app together, but package names make the app easier to scan.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Acceptance Criteria

- [x] Manager and execution orchestration live in `internal/service`.
- [x] Cleanup startup lives in a clear application-level home.
- [x] Production code no longer imports `core`, and the package is removed.

## QA

1. Run the full test suite and confirm it passes.
2. Confirm production code no longer imports `core`.
3. Read `main.go` and confirm the final wiring remains straightforward: load config, create jobs and service, start cleanup, and register routes.

---

*Appended after execution.*

## Completion

**Built:** Moved manager orchestration, async execution, and cleanup startup into `internal/service`, rewired `main.go`, `api`, and end-to-end wiring to the final package layout, and removed the remaining `core` package.

**Decisions:** Kept the existing `Manager` API and cleanup entry point names so the refactor stayed structural rather than behavioral, and moved the existing orchestration-focused tests into `internal/service` to preserve package-private coverage while matching the new layout.

**Deviations:** Kept the temporary jobs alias shim as a test-only file in `internal/service` so the moved orchestration tests could stay focused on service behavior instead of rewriting every reference during the package removal.

**Files:** Created `internal/service/manager.go`, `internal/service/archive.go`, `internal/service/script.go`, and `internal/service/cleanup.go`; moved tests to `internal/service/manager_test.go`, `internal/service/archive_test.go`, `internal/service/script_test.go`, `internal/service/cleanup_test.go`, `internal/service/helpers_test.go`, and `internal/service/jobs_alias_test.go`; updated `main.go`, `api/handlers.go`, `api/handlers_test.go`, and `e2e_test.go`; removed the old `core` package files.

**Notes for next slice:** This plan is complete; production wiring now reads cleanly as config load, job store/service creation, cleanup startup, and route registration, with all behavior still covered by the full Go test suite.
