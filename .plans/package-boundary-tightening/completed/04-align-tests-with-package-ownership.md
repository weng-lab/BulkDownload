# Slice 4: Align tests with package ownership

## Dependencies

Slice 2: Extend the service request flow to tarball and script.
Slice 3: Stabilize API error mapping.

## Description

Refocus tests so handler tests verify HTTP behavior and error mapping, service tests verify validation and routing, and end-to-end tests continue proving the unchanged lifecycle contract. Reduce tests that depend on the old package-boundary assumptions so the suite reinforces the new architecture.

## Expected Behaviors Addressed

- HTTP handlers stay thin and primarily coordinate request/response flow.
- Job creation validation happens in one place instead of being split between transport and service code.
- Invalid job requests return stable, intentional error responses based on typed service errors.
- The existing job lifecycle remains the same: create, process, poll status, download artifact, expire and clean up.
- The package layout remains simple and concrete without adding unnecessary abstractions.

## Acceptance Criteria

- [x] Handler tests focus on HTTP behavior and error mapping instead of business validation internals.
- [x] Service tests clearly cover validation rules, routing, and typed error behavior.
- [x] End-to-end tests still prove the create/status/download/cleanup flows across supported job types.
- [x] The test suite passes with the new package boundary expectations.

## QA

1. Run the full Go test suite.
2. Confirm handler tests assert transport-only behavior and service error mapping rather than duplicating service validation logic.
3. Confirm service tests cover job type validation, path validation, file existence failures, and dispatcher routing.
4. Confirm end-to-end tests still verify successful create, status polling, artifact download, and cleanup for the supported job flows.

---

*Appended after execution.*

## Completion

**Built:** Refocused create-job handler tests on transport behavior and error mapping, clarified service-level `CreateJob` coverage around routing and typed validation, and extended end-to-end cleanup coverage across zip, tarball, and script jobs.

**Decisions:** Kept the runtime code unchanged because the boundary shift was already implemented in earlier slices; this slice locks the ownership model in with tests that separate API concerns from service concerns.

**Deviations:** No production code changes were needed, so the work stayed entirely in tests and slice bookkeeping rather than forcing additional refactors.

**Files:** Modified `api/handlers_test.go`; modified `internal/service/create_job_test.go`; modified `e2e_test.go`; moved this slice to `completed/04-align-tests-with-package-ownership.md`.

**Notes for next slice:** The package-boundary tightening plan is now fully covered by tests, with handler assertions scoped to HTTP behavior, service assertions scoped to request validation/routing, and end-to-end coverage still protecting lifecycle behavior for all supported job types.
