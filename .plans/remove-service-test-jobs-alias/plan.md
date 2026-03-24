# Remove service test jobs alias

## Problem

The service test suite currently depends on `internal/service/jobs_alias_test.go` to re-export types, constants, errors, and constructors from `internal/jobs` into `package service`. That hides where those symbols come from, adds a test-only indirection layer, and makes the tests feel less idiomatic than they need to be.

## Solution

Remove the alias shim by updating service tests to import `internal/jobs` directly wherever they need job models or store helpers. Keep the change scoped to tests so production behavior stays exactly the same while the test suite becomes more explicit about package ownership.

## Expected Behavior

- Service tests refer to job-related types and constants through explicit `jobs.` qualifiers instead of relying on hidden aliases.
- The service package test suite passes without `internal/service/jobs_alias_test.go`.
- Cleanup, archive, script, and manager tests continue proving the same runtime behavior as before.
- The full Go test suite still passes after the shim is removed.

## Implementation Decisions

- Keep the refactor test-only and avoid changing production package boundaries or runtime code.
- Use direct imports from `internal/jobs` in each affected test file rather than creating a new helper layer.
- Replace unqualified aliases with explicit `jobs.Job`, `jobs.Jobs`, `jobs.JobType`, `jobs.Status*`, `jobs.JobType*`, and `jobs.NewJobs()` references.
- Migrate the smallest test files first, then the archive tests, then the larger manager test file, so verification can happen incrementally.
- Delete the shim only after all remaining consumers have been updated.

## Testing Approach

Use targeted `go test` runs after each slice to catch missed identifier replacements early, then finish with package-level and repo-wide test runs. Good coverage here means proving the refactor did not alter cleanup behavior, archive execution behavior, script-job behavior, or manager dispatch behavior while also confirming the service tests compile without the alias file.

## Out of Scope

- Changing production code in `internal/service` or `internal/jobs`
- Renaming job model types or constants
- Reworking test structure beyond what is needed to remove the alias shim
- Broader package-boundary refactors outside the service test suite
