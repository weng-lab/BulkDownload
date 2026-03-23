# Slice 3: Extract Artifact Builders

## Dependencies

Slice 1: Extract Job Model And Store.

## Description

Move zip, tarball, and script generation helpers into `internal/artifacts`, including progress tracking and artifact file helpers, while keeping the existing dispatch flows and generated outputs unchanged.

## Expected Behaviors Addressed

- Archive and script generation live in one obvious place.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Acceptance Criteria

- [x] Zip, tarball, and script generation logic lives in `internal/artifacts`.
- [x] Progress tracking and artifact cleanup helpers move with the artifact generation code.
- [x] Existing archive and script outputs remain unchanged.

## QA

1. Run archive and script focused tests and confirm they still pass.
2. Run API or end-to-end flows for zip, tarball, and script jobs.
3. Confirm artifact-building code is no longer mixed into the old catch-all package.

---

*Appended after execution.*

## Completion

**Built:** Moved archive creation, script generation, progress tracking, and artifact file cleanup helpers into `internal/artifacts`, then rewired `core.Manager` and cleanup orchestration to use the new package without changing job behavior or generated outputs.

**Decisions:** Kept job execution methods on `core.Manager` so dispatch and lifecycle orchestration stay in place for the later service slice, and exposed only the artifact entry points that orchestration needs (`CreateZipFromRoot`, `CreateTarballFromRoot`, `CreateDownloadScript`, and `CleanupFile`).

**Deviations:** Left the archive integration tests in `core` for now and pointed them at `internal/artifacts` instead of moving every manager-facing test in this slice; only the progress-specific unit tests moved with the package because they depend on package-private helpers.

**Files:** Created `internal/artifacts/archive.go`, `internal/artifacts/script.go`, `internal/artifacts/progress.go`, `internal/artifacts/files.go`, and `internal/artifacts/progress_test.go`; removed `core/progress.go`, `core/files.go`, and `core/progress_test.go`; updated `core/archive.go`, `core/script.go`, `core/cleanup.go`, and `core/archive_test.go`.

**Notes for next slice:** `core` now mainly holds orchestration and cleanup startup; the next slice can extract `Manager` and related lifecycle behavior into `internal/service` without needing to revisit archive or script generation internals.
