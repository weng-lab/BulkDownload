# Slice 1: API Owns Router Assembly

## Dependencies

None.

## Description

Move route and middleware assembly into the `api` package by introducing a router-construction entrypoint there, then switch `main` to consume that router. End-to-end, the service still starts and serves the same `/jobs`, `/status/{id}`, and `/download/{id}` behavior, but route ownership now lives with the API layer instead of the entrypoint.

## Expected Behaviors Addressed

- HTTP endpoints and middleware behavior stay the same.
- Route registration lives with the API package instead of being defined in the entrypoint.

## Acceptance Criteria

- [ ] The `api` package exposes a router builder that wires middleware and the existing handlers.
- [ ] `main` no longer defines the route table directly.
- [ ] Existing handler behavior and route paths remain unchanged.
- [ ] Tests cover that the API router serves the expected endpoints.

## QA

1. Start the app.
2. Send a valid `POST /jobs` request and confirm it is accepted as before.
3. Request `GET /status/{id}` for a created job and confirm the same response behavior as before.
4. Request `GET /download/{id}` for a completed job and confirm download behavior is unchanged.
5. Confirm CORS middleware is still applied to routed requests.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
