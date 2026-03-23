# Slice 2: Extend the service request flow to tarball and script

## Dependencies

Slice 1: Service-owned zip request flow.

## Description

Expand the same service-owned create-job path to cover tarball and script requests while preserving the existing per-type dispatchers underneath. Successful tarball and script creation should behave exactly as before, but all request semantics should now be validated and routed from the service layer instead of the API layer.

## Expected Behaviors Addressed

- Job creation validation happens in one place instead of being split between transport and service code.
- The existing job lifecycle remains the same: create, process, poll status, download artifact, expire and clean up.
- The package layout remains simple and concrete without adding unnecessary abstractions.

## Acceptance Criteria

- [x] Valid tarball and script create-job requests are handled through the same service-owned entry point as zip.
- [x] Tarball and script requests continue to use the existing per-type dispatchers after service validation/routing.
- [x] Invalid tarball and script requests fail through the same typed service validation path used for zip.
- [x] Tarball artifacts and generated scripts remain unchanged in behavior.

## QA

1. Run the relevant tests or start the app.
2. Send a valid tarball `POST /jobs` request and confirm the response is `202 Accepted`, the job reaches `done`, and the downloaded tarball contains the expected files.
3. Send a valid script `POST /jobs` request and confirm the response is `202 Accepted`, the job reaches `done`, and the downloaded script still contains the expected base URL, download root, and file paths.
4. Send invalid tarball and script requests with bad paths or missing files and confirm they return `400 Bad Request` through the shared service validation path.

---

*Appended after execution.*

## Completion

**Built:** Added service, handler, and end-to-end coverage proving tarball and script requests flow through `Manager.CreateJob`, preserve their existing artifact/script outputs, and return `400 Bad Request` for invalid paths through the shared typed service validation path.

**Decisions:** Kept the production implementation unchanged because slice 1 had already centralized tarball and script routing in the service entry point; this slice locks that behavior in with explicit regression coverage instead of layering on duplicate logic.

**Deviations:** No service code changes were required once the existing flow was verified, so the work focused on tests and slice bookkeeping rather than additional runtime refactoring.

**Files:** Modified `api/handlers_test.go`; modified `e2e_test.go`; modified `internal/service/create_job_test.go`; moved this slice to `completed/02-extend-service-request-flow-to-tarball-and-script.md`.

**Notes for next slice:** Error-shape coverage now spans zip, tarball, and script requests, so the API error-mapping cleanup can tighten handler assertions without re-establishing request ownership for each job type.
