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

**Built:** Added `api.NewRouter` so the API package now owns CORS middleware and the `/jobs`, `/status/{id}`, and `/download/{id}` route assembly; updated `main` and the end-to-end test harness to consume that router.

**Decisions:** Kept router construction as a small exported helper in `api/router.go` so handler ownership and route wiring stay together without changing handler contracts or server setup responsibilities in `main`.

**Deviations:** None.

**Files:** `api/router.go`, `api/router_test.go`, `main.go`, `e2e_test.go`, `.plans/simplify-main-router-bootstrap/completed/01-api-owns-router-assembly.md`

**Notes for next slice:** `main` still uses the extracted shutdown helpers (`newHTTPServer`, `serveUntilShutdown`, `newShutdownContext`); the next slice can focus on inlining the bootstrap and graceful-shutdown flow while keeping the new API-owned router entrypoint.
