# Reorganize Project Packages

## Problem

The codebase is small, but the package structure makes it feel more complex than it is. `api` is narrow and easy to understand, while `core` mixes several unrelated responsibilities: job domain types, in-memory state, orchestration, artifact generation, config loading, and cleanup. That makes it harder to navigate the code and harder to see where new logic belongs.

## Solution

Split the current `core` package into a few smaller packages with clear responsibilities, while keeping the existing runtime behavior and API contract the same.

Target package layout:

- `api` for HTTP handlers and transport types
- `internal/config` for env loading, defaults, and config parsing
- `internal/jobs` for job model, statuses and types, in-memory store, and expiration queries
- `internal/artifacts` for zip, tar, and script generation, progress tracking, and artifact file helpers
- `internal/service` for job creation, dispatch, async execution, and lifecycle orchestration
- `main.go` as the composition root and router wiring

This keeps the architecture simple and concrete, without adding interfaces or layered abstractions up front.

## Expected Behavior

- A developer can quickly find job state and storage logic in `internal/jobs`.
- Archive and script generation live in one obvious place.
- HTTP handlers remain focused on request and response flow.
- `main.go` still wires the app together, but package names make the app easier to scan.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Implementation Decisions

- Remove `core` over time rather than renaming it and keeping the same ambiguity.
- Keep `api` dependent on concrete types at first.
- Extract the cleanest seams first: jobs, then config, then artifacts, then orchestration.
- Keep `internal/service` thin so it does not become a second catch-all package.
- Avoid introducing interfaces until there is a clear use case.
- Preserve current tests and behavior as much as possible during the move.
- Likely file mapping:
- `internal/jobs` absorbs the current job model, job status and type definitions, job errors, in-memory store, and expiration logic.
- `internal/config` absorbs the current config and env-loading behavior.
- `internal/artifacts` absorbs the progress helper, artifact file cleanup helper, and the pure archive and script generation logic.
- `internal/service` absorbs manager and execution orchestration, plus cleanup startup if it remains an application lifecycle concern.

## Testing Approach

Use the current tests as migration safety rails instead of rewriting them early. Preserve end-to-end behavior as the main proof that the refactor did not change the API contract. Move package-specific tests alongside the code when practical. Expect some churn where tests currently rely on package-private helpers, but avoid redesigning the tests unless that becomes necessary to support the new package boundaries.

## Out of Scope

- API behavior changes
- Persistent storage
- Worker queues
- Interface-driven refactors
- Major router or bootstrap redesign
- Feature changes unrelated to package organization
