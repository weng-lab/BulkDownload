# Slice 2: Main Becomes Linear Bootstrap And Shutdown Flow

## Dependencies

- Slice 1: API Owns Router Assembly

## Description

Reshape `main` so it reads top-to-bottom like the desired sketch: load config, create the jobs directory, build dependencies, start cleanup, build the API router, configure the HTTP server, start serving, wait for shutdown signals, and shut down gracefully. End-to-end, startup and shutdown behavior remain intact, but the extra serve/shutdown helper layer is removed and the entrypoint becomes the clear orchestration path.

## Expected Behaviors Addressed

- The application still starts the same way from the user's point of view.
- `main` becomes easy to scan and mostly reads as setup + serving + shutdown.
- Graceful shutdown still stops cleanup and shuts the server down cleanly when the process receives a termination signal.

## Acceptance Criteria

- [ ] `main` inlines the startup and shutdown orchestration instead of calling lifecycle helpers.
- [ ] Server timeout configuration remains explicit and unchanged in behavior.
- [ ] Cleanup is started during boot and stopped during shutdown/process exit.
- [ ] Tests are updated to cover the new shape without relying on removed helpers.
- [ ] App startup, serving, and graceful shutdown still work from an external caller’s point of view.

## QA

1. Start the app and confirm it logs startup and begins listening on the configured port.
2. Create a job and check status to verify the app serves requests normally after the refactor.
3. Send `SIGINT` or `SIGTERM` to the process.
4. Confirm the app begins graceful shutdown instead of exiting abruptly.
5. Confirm the process exits cleanly and does not leave the server hanging.

---

*Appended after execution.*

## Completion

**Built:** Inlined the server bootstrap and graceful-shutdown flow inside `run`, so startup now reads as directory setup, dependency construction, cleanup start, router/server setup, serve, and shutdown; updated `main` to register signals directly and refreshed tests around the new orchestration shape.

**Decisions:** Kept `run` as the single testable orchestration entrypoint while removing the extra lifecycle helpers; introduced small package-level hooks for serving, shutdown, cleanup, and signal registration so tests can validate timeout config and shutdown behavior without reintroducing the old helper layer.

**Deviations:** None.

**Files:** `main.go`, `main_test.go`, `.plans/simplify-main-router-bootstrap/completed/02-main-becomes-linear-bootstrap-and-shutdown-flow.md`

**Notes for next slice:** No further slices are pending in `simplify-main-router-bootstrap`; the plan's bootstrap/router refactor is now complete.
