# Manager And Jobs Lifecycle API Refinement

## Problem

The current split between `Manager` and `Jobs` mostly works, but the mutation API is too generic for the job model we actually support. `Manager` currently carries several thin helpers that all funnel into `Jobs.Update`, which makes the store API broader and less intention-revealing than it needs to be. At the same time, the explicit dispatch methods on `Manager` are worth keeping because they make the public orchestration surface easy to read.

## Solution

Keep `Manager` responsible for creating and dispatching jobs, and keep the explicit per-job-type dispatch methods. Refactor `Jobs` so it exposes only the supported lifecycle transitions through named store methods, and remove the generic `Update` entrypoint entirely. Let the named store operations express the allowed state changes directly, while `Manager` continues to coordinate when those transitions happen.

## Expected Behavior

- Creating a job still goes through the explicit manager dispatch methods for zip, tarball, and script jobs.
- A new job is still assembled by `Manager` with its ID, type, expiry, and file list before being stored.
- Job execution paths update state through explicit store lifecycle methods instead of a generic mutation callback.
- Callers can only perform the supported status/progress transitions, not arbitrary in-place job edits.
- Existing status polling and download behavior remain unchanged from a user perspective.

## Implementation Decisions

- Keep `Manager` as the orchestration layer and preserve the readable public dispatch API.
- Keep initial `Job` construction in `Manager`; do not move creation semantics into `Jobs`.
- Replace `Jobs.Update` with explicit methods that match the supported lifecycle transitions:
  - `MarkProcessing`
  - `SetProgress`
  - `MarkFailed`
  - `MarkDone`
- Allow `Jobs` to use a private internal helper if needed to avoid lock/mutation duplication, but do not expose any generic update method publicly.
- Keep the manager-level helper methods only if they still improve readability after the store API becomes explicit; otherwise let execution paths call the named `Jobs` methods directly.
- Treat the old `Add` then `Get` concern as resolved already and do not include extra work for it.

## Testing Approach

- Update existing store tests to verify each named lifecycle method mutates only the intended fields and preserves snapshot semantics from `Get`.
- Replace current `Jobs.Update` coverage with focused tests for processing, progress clamping, failure, and completion transitions.
- Keep manager and archive/script tests centered on externally visible behavior: job creation, retries on duplicate IDs, successful completion, and failure handling.
- Verify end-to-end behavior remains the same for create, status, and download flows.

## Out of Scope

- Collapsing the explicit manager dispatch API into a single generic public dispatch method.
- Moving job creation or ID/expiry generation into `Jobs`.
- Changing API request/response shapes or user-facing job semantics.
- Broader redesign of the store beyond the lifecycle-specific mutation surface.
