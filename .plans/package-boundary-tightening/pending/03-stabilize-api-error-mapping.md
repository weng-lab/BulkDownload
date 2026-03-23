# Slice 3: Stabilize API error mapping

## Dependencies

Slice 1: Service-owned zip request flow.

## Description

Make the API layer consistently translate minimal typed service errors into stable frontend-facing HTTP responses, while keeping malformed JSON and missing transport fields handled locally in `api`. This locks in the new boundary and removes string-based decision making from handler logic where practical.

## Expected Behaviors Addressed

- HTTP handlers stay thin and primarily coordinate request/response flow.
- Invalid job requests return stable, intentional error responses based on typed service errors.

## Acceptance Criteria

- [ ] Malformed JSON and missing transport fields still return `400 Bad Request` directly from `api`.
- [ ] Typed service validation errors are mapped predictably to `400 Bad Request` without brittle string matching in handlers.
- [ ] Unexpected service failures still map to `500 Internal Server Error`.
- [ ] Handler logic is simpler and focused on transport-to-service translation.

## QA

1. Run the handler tests or exercise the API manually.
2. Send malformed JSON to `POST /jobs` and confirm the API still returns `400 Bad Request` before reaching business validation.
3. Send a request with missing required fields and confirm the API still returns `400 Bad Request`.
4. Send requests that fail service validation and confirm they return stable `400 Bad Request` responses.
5. Force or simulate an unexpected service failure and confirm the API returns `500 Internal Server Error`.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
