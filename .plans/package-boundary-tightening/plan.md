# Tighten Package Boundaries

## Problem

The package reorganization improved the layout, but responsibilities are still a little blurry. The `api` package still performs validation and some business-adjacent decision making, which makes it harder to keep transport concerns separate from application logic. I want each package to have a tighter, more obvious scope so it is clear where new logic belongs and easier to understand what can change without affecting the API layer.

## Solution

Move job request validation and dispatch selection fully into the service layer while keeping the current package structure. `api` remains responsible for HTTP concerns only: decoding requests, extracting route params, invoking service methods, and translating service errors into HTTP responses. `service` becomes the clear owner of job creation rules, path/file validation, dispatch routing, async execution, and cleanup lifecycle. `jobs`, `artifacts`, and `config` keep their current focused roles.

## Expected Behavior

- HTTP handlers stay thin and primarily coordinate request/response flow.
- Job creation validation happens in one place instead of being split between transport and service code.
- Invalid job requests return stable, intentional error responses based on typed service errors.
- The existing job lifecycle remains the same: create, process, poll status, download artifact, expire and clean up.
- The package layout remains simple and concrete without adding unnecessary abstractions.

## Implementation Decisions

- Keep the current package layout rather than reorganizing packages again.
- Keep `api` request and response structs, and continue reusing domain enums such as job type and job status in transport responses.
- Keep cleanup ownership in the service layer because it is application lifecycle orchestration.
- Keep the existing per-type dispatchers for zip, tarball, and script jobs.
- Add a small service-level entry point that validates input, chooses the appropriate dispatcher, and returns typed errors for invalid frontend requests.
- Move job type validation, path validation, and file existence checks into the service layer.
- Limit `api` validation to transport-local concerns such as malformed JSON, missing body fields, and missing route parameters.
- Use minimal typed errors in the service layer so the API can map failures to HTTP status codes without relying on brittle string matching.
- Keep `service` thin by extracting shared validation/routing helpers instead of pushing all logic directly into each dispatcher.
- Preserve current runtime behavior and API contract as much as possible while shifting ownership of validation.

## Testing Approach

Use the current tests as safety rails and adjust them to reflect the new ownership boundaries rather than rewriting coverage from scratch. Handler tests should verify transport behavior and HTTP error mapping, not business validation internals. Service tests should cover job type validation, path rules, file existence failures, dispatcher routing, and typed error behavior. End-to-end tests should continue proving that the full job lifecycle and API contract still work unchanged for valid requests and expected invalid requests.

## Out of Scope

- Changing routes or the public API shape
- Changing job lifecycle behavior or artifact outputs
- Introducing interfaces or a larger abstraction layer
- Changing persistence strategy or adding queues/workers
- Redesigning `main.go` beyond normal wiring updates
- Duplicating domain enums into transport-specific API types
