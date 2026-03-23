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

- [ ] Valid tarball and script create-job requests are handled through the same service-owned entry point as zip.
- [ ] Tarball and script requests continue to use the existing per-type dispatchers after service validation/routing.
- [ ] Invalid tarball and script requests fail through the same typed service validation path used for zip.
- [ ] Tarball artifacts and generated scripts remain unchanged in behavior.

## QA

1. Run the relevant tests or start the app.
2. Send a valid tarball `POST /jobs` request and confirm the response is `202 Accepted`, the job reaches `done`, and the downloaded tarball contains the expected files.
3. Send a valid script `POST /jobs` request and confirm the response is `202 Accepted`, the job reaches `done`, and the downloaded script still contains the expected base URL, download root, and file paths.
4. Send invalid tarball and script requests with bad paths or missing files and confirm they return `400 Bad Request` through the shared service validation path.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
