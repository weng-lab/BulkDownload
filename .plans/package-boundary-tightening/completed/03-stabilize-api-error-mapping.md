# Slice 3: Stabilize API error mapping

## Dependencies

Slice 1: Service-owned zip request flow.

## Description

Make the API layer consistently translate minimal typed service errors into stable frontend-facing HTTP responses, while keeping malformed JSON and missing transport fields handled locally in `api`. This locks in the new boundary and removes string-based decision making from handler logic where practical.

## Expected Behaviors Addressed

- HTTP handlers stay thin and primarily coordinate request/response flow.
- Invalid job requests return stable, intentional error responses based on typed service errors.

## Acceptance Criteria

- [x] Malformed JSON and missing transport fields still return `400 Bad Request` directly from `api`.
- [x] Typed service validation errors are mapped predictably to `400 Bad Request` without brittle string matching in handlers.
- [x] Unexpected service failures still map to `500 Internal Server Error`.
- [x] Handler logic is simpler and focused on transport-to-service translation.

## QA

1. Run the handler tests or exercise the API manually.
2. Send malformed JSON to `POST /jobs` and confirm the API still returns `400 Bad Request` before reaching business validation.
3. Send a request with missing required fields and confirm the API still returns `400 Bad Request`.
4. Send requests that fail service validation and confirm they return stable `400 Bad Request` responses.
5. Force or simulate an unexpected service failure and confirm the API returns `500 Internal Server Error`.

---

*Appended after execution.*

## Completion

**Built:** Extracted create-job error handling into a dedicated API helper so handler flow stays focused on decoding, calling `service`, and writing responses, then added regression tests for transport-local `400` responses, typed service validation `400` responses, and unexpected `500` failures.

**Decisions:** Kept the service error surface minimal and reused the existing typed request error, with the API owning a single helper that maps request errors to `400 Bad Request` and everything else to a generic `500 Internal Server Error`.

**Deviations:** No service-package refactor was needed because the typed error contract from slice 1 already supported stable mapping; this slice tightened the boundary by simplifying handler code and locking the mapping behavior in with tests.

**Files:** Modified `api/handlers.go`; modified `api/handlers_test.go`.

**Notes for next slice:** Create-job boundary coverage now explicitly separates transport validation from service validation and unexpected failures, so test-alignment work can focus on trimming overlapping assertions and reinforcing package ownership.
