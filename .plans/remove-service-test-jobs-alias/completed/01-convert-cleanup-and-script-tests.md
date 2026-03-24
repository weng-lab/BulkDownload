# Slice 1: Convert cleanup and script tests

## Dependencies

None.

## Description

Replace the test-only `service` package aliases with explicit `internal/jobs` imports in the smallest service test files so those tests compile and pass without relying on `jobs_alias_test.go`.

## Expected Behaviors Addressed

- Service tests use explicit package ownership for job models and store helpers.
- Cleanup and script job tests continue proving the same lifecycle behavior after the alias shim is removed.
- Test readability improves by making cross-package dependencies visible at the call site.

## Acceptance Criteria

- [ ] `internal/service/cleanup_test.go` imports `github.com/jair/bulkdownload/internal/jobs` directly.
- [ ] `internal/service/script_test.go` imports `github.com/jair/bulkdownload/internal/jobs` directly.
- [ ] All former alias-based references in those files use explicit `jobs.` qualifiers.
- [ ] Targeted tests pass without introducing behavioral changes.

## QA

1. Review `internal/service/cleanup_test.go` and confirm `Job`, `Status*`, `JobType*`, and `NewJobs()` references now use `jobs.` explicitly.
2. Review `internal/service/script_test.go` and confirm `Job` and `JobTypeScript` references now use `jobs.` explicitly.
3. Run `go test ./internal/service -run 'TestSweepExpired|TestStartCleanup_SweepsOnTick|TestManagerExecuteScriptJob'`.
4. Confirm the tests still verify the same cleanup and script-job behavior as before.

---

*Appended after execution.*

## Completion

**Built:** Converted `internal/service/cleanup_test.go` and `internal/service/script_test.go` to import `internal/jobs` directly and use explicit jobstore-qualified types, statuses, job types, and store construction.

**Decisions:** Used a `jobstore` import alias so the tests stay readable while avoiding local variable shadowing with the jobs store instances.

**Deviations:** No functional deviations; the only implementation detail beyond the plan was renaming local store variables in `cleanup_test.go` to avoid colliding with the imported package alias.

**Files:** Modified `internal/service/cleanup_test.go`, `internal/service/script_test.go`, and this slice record.

**Notes for next slice:** `jobs_alias_test.go` is still required by the remaining archive and manager tests; this slice only removes the cleanup/script test dependencies on the shim.
