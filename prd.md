# PRD: BulkDownload Reliability and Complexity Reduction

## Problem Statement

BulkDownload currently works for its main happy paths, but the service is more mentally expensive to change than its product scope justifies. The same request flow is implemented multiple times for zip, tarball, and script jobs. Validation rules are split across HTTP handlers and core job processing. Cleanup runs in the background without clear lifecycle ownership. Tests rely heavily on timing and polling, which makes failures harder to trust.

From the user's perspective, this creates a product risk: behavior is harder to predict, configuration mistakes are easier to make, request rules are not always consistent across endpoints, and small feature or maintenance changes cost more engineering time than they should. The goal is to make the service easier to understand, safer to operate, and cheaper to extend without changing the fundamental product offering.

## Solution

Refactor BulkDownload so that job validation, job lifecycle, request handling, cleanup, documentation, and testing all follow one coherent end-to-end contract. The service should preserve its current core behavior — create archive jobs, create script jobs, expose status, and serve artifacts — while reducing duplication and making each behavior owned in one place.

From the user's perspective, the result should be a service whose endpoints behave consistently, whose documented request rules match reality, whose jobs do not disappear mid-processing, whose configuration fails fast when invalid, and whose implementation can be modified confidently because tests verify externally visible behavior rather than timing accidents.

## User Stories

1. As an API client, I want zip job requests to follow one clear validation contract, so that I know immediately whether my request is accepted.
2. As an API client, I want tarball job requests to follow the same validation rules as zip jobs where applicable, so that I do not have to memorize endpoint-specific surprises.
3. As an API client, I want script job requests to document and enforce their own path rules clearly, so that I can prepare requests correctly the first time.
4. As an API client, I want invalid file lists to be rejected before a job is stored, so that I do not see jobs accepted and then fail later for predictable reasons.
5. As an API client, I want create endpoints to return a consistent response shape, so that client code is simple and reusable.
6. As an API client, I want status responses to follow a stable contract, so that I can build polling behavior with confidence.
7. As an API client, I want download behavior to be consistent across job types, so that completed jobs feel predictable.
8. As an API client, I want error responses to reflect structured request problems instead of brittle wording, so that my client can respond to failures reliably.
9. As an API client, I want route behavior to match the router configuration, so that nested or refactored routes do not accidentally break handlers.
10. As an operator, I want cleanup behavior to avoid deleting in-flight jobs, so that users do not lose active work.
11. As an operator, I want background cleanup to have explicit lifecycle ownership, so that startup and shutdown behavior is understandable.
12. As an operator, I want invalid cleanup timing values to fail at startup, so that the service does not panic later in production.
13. As an operator, I want invalid TTL values to fail fast, so that job expiry behavior is trustworthy.
14. As an operator, I want documented config options to match actual supported config, so that deployment setup is low-friction.
15. As an operator, I want helper scripts and README examples to match current request rules, so that local validation and demos do not teach the wrong behavior.
16. As a maintainer, I want job validation rules owned in one deep module, so that I can change them in one place.
17. As a maintainer, I want the manager to enforce core job invariants, so that non-HTTP callers cannot create broken jobs.
18. As a maintainer, I want zip and tarball processing to share one execution pattern, so that fixes to lifecycle and error handling apply everywhere.
19. As a maintainer, I want script job processing to align with the same job lifecycle model, so that behavior is not split by implementation style.
20. As a maintainer, I want job state transitions to be explicit, so that I can reason about when status, progress, filename, and errors are valid.
21. As a maintainer, I want cleanup to operate against explicit job states, so that retention rules are safe and reviewable.
22. As a maintainer, I want progress updates and job-state mutations to surface meaningful failures, so that silent drift between memory and disk is minimized.
23. As a maintainer, I want archive finalization errors to be surfaced correctly, so that a job is not marked successful when the artifact is incomplete.
24. As a maintainer, I want the codebase to remove no-op wrappers and dead indirection, so that I spend less time jumping between tiny helpers.
25. As a maintainer, I want the HTTP layer to focus on transport concerns, so that domain rules are not scattered through handlers.
26. As a maintainer, I want end-to-end tests to use the same router stack as production, so that passing tests actually validate the deployed path.
27. As a maintainer, I want tests to avoid sleep-heavy polling where deterministic seams are possible, so that CI failures are easier to trust.
28. As a maintainer, I want config tests to avoid depending on global process state when possible, so that parallelism and isolation are safer.
29. As a maintainer, I want security-sensitive path normalization rules tested directly, so that regressions are isolated quickly.
30. As a maintainer, I want generated script behavior covered by focused external-behavior tests, so that quoting and file-list handling remain safe.
31. As a maintainer, I want README examples to reflect the same contracts as the tests, so that product understanding is consistent across code and docs.
32. As a future contributor, I want the service structure to be obvious from the top-level flow, so that onboarding takes minutes instead of hours.
33. As a future contributor, I want to add a new job type without cloning an entire endpoint and processing pipeline, so that extension cost stays low.
34. As a reviewer, I want changes to one job behavior to touch fewer places, so that code review is faster and less error-prone.
35. As a reviewer, I want tests to describe behavior instead of implementation details, so that refactors can be approved with confidence.
36. As a user of the generated scripts, I want download scripts to remain compatible with current usage patterns, so that cleanup work does not break downstream workflows.
37. As a team maintaining the service, I want reliability improvements delivered without changing the fundamental product scope, so that users get safer behavior without relearning the tool.

## Implementation Decisions

- The feature is a reliability and simplification initiative, not a new end-user product feature.
- The service will continue supporting three job types: zip, tarball, and download script.
- Core job validation will move behind a core-owned contract rather than living primarily in the HTTP layer.
- Archive request normalization, source-root enforcement, and invalid-path rejection will be expressed once and reused by every caller.
- Script job validation will be explicit and intentionally documented, rather than being inferred from handler-only behavior.
- The HTTP layer will be narrowed to request decoding, response encoding, and mapping transport-level failures to stable API behavior.
- Route parameter extraction will use the router’s supported mechanism rather than manual string trimming.
- Create, status, and download endpoints will follow one consistent API contract for success and failure handling.
- Job creation will reject invalid inputs before persisting jobs to in-memory storage.
- The job lifecycle will be modeled explicitly around pending, processing, done, and failed states, with clear rules for when progress, artifact name, and error details may be set.
- Cleanup will respect job state and expiration together, instead of deleting jobs based only on timestamp.
- Background cleanup will run under explicit lifecycle ownership so it can be started, stopped, and tested predictably.
- Zip and tarball processing will share a common execution template for state transitions, artifact creation, failure cleanup, and completion.
- Archive writer finalization and other deferred-close paths will be treated as meaningful outcomes, not ignored side effects.
- Progress reporting will remain externally visible but will be governed by a clearer contract about when updates occur and when failures are surfaced.
- Configuration loading will remain environment-driven, but invalid timing values will fail fast and supported options will be simplified where possible.
- Legacy or drifted configuration semantics will only be preserved when they have a clear compatibility reason.
- Documentation and helper scripts will be updated to match the actual request contracts that the server enforces.
- The design should favor deep modules with small, stable interfaces, especially around request validation, job lifecycle management, and app lifecycle management.
- The implementation should reduce shallow wrappers whose only effect is forwarding parameters or renaming behavior without hiding complexity.

## Testing Decisions

- A good test verifies externally visible behavior and contracts, not internal helper structure or incidental wording.
- The most important tests are the ones that prove request validation, job state transitions, cleanup safety, artifact availability, and router-integrated behavior.
- The modules that should receive focused test coverage are the job validation contract, job lifecycle/state transitions, cleanup lifecycle behavior, HTTP contract behavior, archive processing behavior, and script generation behavior.
- Tests for config should focus on supported inputs, invalid inputs, and observable startup outcomes rather than incidental global side effects.
- Integration tests should exercise the same router stack and request flow used in production.
- Timing-heavy polling should be reduced in favor of deterministic seams or explicit lifecycle controls where practical.
- Direct tests should exist for security-sensitive and contract-heavy behaviors, especially path normalization and script quoting semantics.
- Prior art in the repo includes current handler tests, archive tests, config tests, cleanup tests, and end-to-end lifecycle tests; those should be tightened and reused where they already express external behavior well.
- Refactors should preserve or improve behavior coverage while deleting redundant tests that merely restate the same happy path through multiple layers.

## Out of Scope

- Adding new end-user job types.
- Persisting job metadata beyond in-memory storage.
- Changing the fundamental artifact formats or download model.
- Replacing the generated shell script approach with a different client distribution mechanism.
- Building a frontend UI.
- Expanding deployment automation beyond aligning current docs and helper scripts.
- Large-scale performance optimization work unrelated to the current reliability and maintainability goals.

## Further Notes

- This PRD is intentionally centered on reducing mental tax and operational ambiguity in a small service.
- Success should be measured by clearer contracts, safer lifecycle behavior, smaller change surfaces for common fixes, and more trustworthy tests.
- The desired outcome is not abstraction for its own sake; it is fewer places to look, fewer chances for drift, and simpler end-to-end reasoning.
