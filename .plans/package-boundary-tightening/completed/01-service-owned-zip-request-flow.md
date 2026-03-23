# Slice 1: Service-owned zip request flow

## Dependencies

None.

## Description

Build a single service entry point for create-job requests and move zip request semantics into `service`. The API layer should keep handling transport concerns such as JSON decoding, required field presence, and HTTP response writing, while the service layer owns supported type checks, path rules, file existence validation, and dispatch selection for zip requests.

## Expected Behaviors Addressed

- HTTP handlers stay thin and primarily coordinate request/response flow.
- Job creation validation happens in one place instead of being split between transport and service code.
- Invalid job requests return stable, intentional error responses based on typed service errors.

## Acceptance Criteria

- [x] A valid zip create-job request is routed through a service-owned entry point and still returns `202 Accepted`.
- [x] Zip request validation for supported type, path rules, and file existence lives in `service` instead of `api`.
- [x] Invalid zip requests produce typed service errors that the API maps to `400 Bad Request`.
- [x] The zip job lifecycle still works end-to-end: create, process, poll status, and download artifact.

## QA

1. Start the app or run the relevant handler/service tests.
2. Send a valid `POST /jobs` request with `{"type":"zip","files":["nested/alpha.txt"]}` and confirm the response is `202 Accepted` with a job id.
3. Poll `GET /status/{id}` until the job completes, then call `GET /download/{id}` and confirm the zip downloads successfully.
4. Send invalid zip requests with an unsupported type, an absolute path, and a missing file, and confirm each returns `400 Bad Request` through the service validation path.

## Completion

**Built:** Added a service-owned `CreateJob` entry point that validates request type and files, dispatches zip jobs through `service`, and lets the API map typed request errors back to `400 Bad Request` while preserving the existing zip lifecycle.

**Decisions:** Used a single typed `CreateJobRequestError` with stable user-facing messages so the API can distinguish validation failures from internal dispatch failures without string matching in the handler.

**Deviations:** The new service entry point also routes tarball and script requests through the same validation path so request ownership stays centralized instead of splitting validation by type in `api`.

**Files:** Modified `api/handlers.go`; created `internal/service/create_job.go`; created `internal/service/create_job_test.go`.

**Notes for next slice:** Tarball and script requests already enter through `Manager.CreateJob`, so the next slice can focus on any type-specific behavior and broader error-mapping cleanup without moving more transport validation code.
