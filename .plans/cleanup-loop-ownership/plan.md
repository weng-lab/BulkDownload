# Cleanup Loop Ownership

## Problem

I want the cleanup ticker to have explicit ownership so its lifetime is easier to reason about in tests. Right now `StartCleanup` launches a background goroutine and never gives the caller a way to stop it, which makes the lifecycle implicit and leaves ticker-based tests relying on time and process exit rather than bounded cleanup.

## Solution

I will keep the existing process-scoped cleanup model, but make ownership explicit by having the startup path return a stop hook for the cleanup loop. This avoids introducing `context.Context` just for one background ticker while still giving tests a deterministic way to end the goroutine they started.

## Expected Behavior

- Starting periodic cleanup also gives the caller a way to stop it explicitly.
- The service can keep using cleanup as a long-lived background task without changing its runtime model.
- Tests that start the cleanup loop can stop it before the test ends instead of depending on process lifetime.
- Cleanup behavior remains centered on the existing sweep logic; only lifecycle control becomes more explicit.

## Implementation Decisions

- Treat this as an ownership and testability improvement, not as a production goroutine leak bug.
- Do not introduce `context.Context` solely for this ticker, since the service does not otherwise use context-based lifecycle management.
- Return a `stop func()` from the cleanup startup API as the minimal lifecycle contract.
- Keep the periodic loop thin and preserve the existing separation between "run one sweep" and "schedule repeated sweeps."
- Favor explicit caller ownership: the code that starts cleanup is also responsible for stopping it when that matters, especially in tests.

## Testing Approach

- Keep most behavioral coverage on the one-shot sweep logic, since that is the core cleanup behavior.
- Update ticker-based tests to capture the returned stop hook and stop the loop before the test exits.
- Verify the periodic cleanup path still removes expired jobs and artifacts on tick.
- Prefer tests that bound goroutine lifetime explicitly rather than relying on process shutdown or unbounded background work.

## Out of Scope

- Introducing app-wide graceful shutdown orchestration.
- Converting background lifecycle management to `context.Context`.
- Expanding cleanup responsibilities beyond the current periodic sweep behavior.
- Reworking unrelated job lifecycle or deletion semantics.
