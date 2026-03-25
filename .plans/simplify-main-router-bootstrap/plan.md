# Simplify Main And Move Router Ownership Into API

## Problem

I don't like the current shape of startup in `main`. It still owns route wiring, which makes the entrypoint feel more responsible for HTTP details than it should be. I want `main` to read like a straightforward bootstrap-and-serve flow, while the API package owns how routes are assembled.

## Solution

Move router construction into the API layer and reshape `main` so it reads top-to-bottom like the sketch: load config, create required directories, build dependencies, start cleanup, build the router, configure the server, start serving, wait for shutdown, and stop gracefully. Drop the extra lifecycle helper layer and keep the flow inlined in `main`.

## Expected Behavior

- The application still starts the same way from the user's point of view.
- HTTP endpoints and middleware behavior stay the same.
- `main` becomes easy to scan and mostly reads as setup + serving + shutdown.
- Route registration lives with the API package instead of being defined in the entrypoint.
- Graceful shutdown still stops cleanup and shuts the server down cleanly when the process receives a termination signal.

## Implementation Decisions

- The API package will expose a router-construction function that returns the fully wired HTTP handler/router.
- Route definitions and HTTP middleware belong to the API layer, alongside the handlers they compose.
- `main` will directly orchestrate startup and shutdown instead of calling a separate serve helper.
- Server configuration stays explicit in `main` so the entrypoint shows the full serving setup in one place.
- Cleanup lifecycle remains tied to process lifecycle, with shutdown behavior handled in the same linear flow as server shutdown.
- The refactor is structural only; it does not change handler contracts, job creation flow, download flow, or configuration semantics.

## Testing Approach

- Keep the existing handler-level tests as the main coverage for endpoint behavior.
- Add focused tests for router construction so the API package proves the expected routes and middleware are wired.
- Update or replace current main-package tests that target extracted lifecycle helpers, since those helpers will no longer exist.
- Keep coverage for server timeout configuration where it still provides value.
- Use an application-level smoke or integration test to verify the refactor did not change the externally visible HTTP behavior.

## Out of Scope

- Changing endpoint paths, request/response shapes, or API semantics.
- Reworking cleanup behavior beyond what is needed to fit the new startup/shutdown flow.
- Introducing a new application container, dependency injection layer, or broader package reorganization.
- Changing CORS policy or other middleware behavior as part of this refactor.
