# Slice 4: Extract Service And Cleanup, Remove Core

## Dependencies

- Slice 1: Extract Job Model And Store.
- Slice 2: Extract Runtime Config.
- Slice 3: Extract Artifact Builders.

## Description

Move manager orchestration and cleanup startup into `internal/service`, wire `api` and `main.go` to the final package layout, and remove the remaining `core` package without changing the service's public behavior.

## Expected Behaviors Addressed

- HTTP handlers remain focused on request and response flow.
- `main.go` still wires the app together, but package names make the app easier to scan.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Acceptance Criteria

- [ ] Manager and execution orchestration live in `internal/service`.
- [ ] Cleanup startup lives in a clear application-level home.
- [ ] Production code no longer imports `core`, and the package is removed.

## QA

1. Run the full test suite and confirm it passes.
2. Confirm production code no longer imports `core`.
3. Read `main.go` and confirm the final wiring remains straightforward: load config, create jobs and service, start cleanup, and register routes.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
