# Slice 3: Convert manager tests and delete shim

## Dependencies

Slice 1: Convert cleanup and script tests.
Slice 2: Convert archive tests.

## Description

Finish the migration by converting the remaining manager tests to direct `internal/jobs` usage, then remove `internal/service/jobs_alias_test.go` entirely and verify the full suite still passes.

## Expected Behaviors Addressed

- Service manager tests use explicit package ownership everywhere instead of hidden test aliases.
- The service package test suite compiles and passes without `jobs_alias_test.go`.
- Repository behavior remains unchanged; only test indirection is removed.

## Acceptance Criteria

- [ ] `internal/service/manager_test.go` imports `github.com/jair/bulkdownload/internal/jobs` directly.
- [ ] All remaining alias-based references in manager tests use explicit `jobs.` qualifiers.
- [ ] `internal/service/jobs_alias_test.go` is deleted.
- [ ] `go test ./internal/service` passes.
- [ ] `go test ./...` passes.

## QA

1. Review `internal/service/manager_test.go` and confirm all job models, store constructors, job types, and statuses come from `jobs.` explicitly.
2. Confirm no tests in `internal/service` reference the old unqualified alias names.
3. Delete `internal/service/jobs_alias_test.go`.
4. Run `go test ./internal/service`.
5. Run `go test ./...`.
6. Confirm the only change is explicit test imports and the alias shim removal, with no production behavior changes.

---

*Appended after execution.*

## Completion

**Built:** Converted `internal/service/manager_test.go` to import `internal/jobs` directly, replaced the remaining alias-based manager test references with explicit `jobstore.` qualifiers, deleted `internal/service/jobs_alias_test.go`, and verified both `go test ./internal/service` and `go test ./...` still pass.

**Decisions:** Kept the existing `jobstore` import alias used by earlier slices so manager tests stay readable while making package ownership explicit at each job-related call site.

**Deviations:** No behavioral deviations; the only extra verification beyond the written QA steps was a repo search to confirm no `internal/service` tests still rely on the old unqualified alias names.

**Files:** Modified `internal/service/manager_test.go`, deleted `internal/service/jobs_alias_test.go`, and moved this slice record to `completed/`.

**Notes for next slice:** No further slices remain for this plan; the alias shim has been fully removed from the service test suite.
