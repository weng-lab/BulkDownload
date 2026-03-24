# Slice 2: Convert archive tests

## Dependencies

Slice 1: Convert cleanup and script tests.

## Description

Update the archive-focused service tests to use `internal/jobs` directly, including helper signatures and expected job values, so archive behavior remains covered without the test-only alias layer.

## Expected Behaviors Addressed

- Archive service tests use explicit package ownership for job types, job records, and statuses.
- Zip and tarball execution tests continue proving successful completion and failure handling.
- Helper functions in archive tests remain aligned with the real source package for job state.

## Acceptance Criteria

- [ ] `internal/service/archive_test.go` imports `github.com/jair/bulkdownload/internal/jobs` directly.
- [ ] All alias-based archive test references use `jobs.Job`, `jobs.JobType`, `jobs.Jobs`, and `jobs.Status*`.
- [ ] Archive helper functions compile without `jobs_alias_test.go`.
- [ ] Targeted archive tests pass without changing archive behavior.

## QA

1. Review `internal/service/archive_test.go` and confirm all job-related types and constants are explicitly qualified with `jobs.`.
2. Run `go test ./internal/service -run 'TestCreateArchive|TestManagerExecuteArchiveJob'`.
3. Confirm zip and tarball success cases still produce completed jobs and artifacts.
4. Confirm missing-file cases still produce failed jobs and no output artifact.

---

*Appended after execution.*

## Completion

**Built:** Converted `internal/service/archive_test.go` to import `internal/jobs` directly and use explicit jobstore-qualified job types, statuses, structs, and helper signatures throughout the archive test coverage.

**Decisions:** Reused the existing `jobstore` alias pattern from the earlier slice so archive tests stay readable while making package ownership explicit at each job-related call site.

**Deviations:** No behavioral deviations; the only notable implementation detail was keeping `jobs_alias_test.go` in place because the remaining manager-focused tests still depend on it until the final slice.

**Files:** Modified `internal/service/archive_test.go` and moved this slice record to `completed/`.

**Notes for next slice:** The remaining work is concentrated in manager-oriented service tests plus deleting `internal/service/jobs_alias_test.go` once those references are converted.
