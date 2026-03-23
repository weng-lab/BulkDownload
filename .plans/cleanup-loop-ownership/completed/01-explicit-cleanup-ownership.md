# Slice 1: Explicit Cleanup Ownership

## Dependencies

None.

## Description

Make periodic cleanup explicitly owned by the caller by returning a stop hook from cleanup startup, keep the existing runtime behavior intact, and update ticker-based tests to stop the background loop they start.

## Expected Behaviors Addressed

- Starting periodic cleanup also gives the caller a way to stop it explicitly.
- The service can keep using cleanup as a long-lived background task without changing its runtime model.
- Tests that start the cleanup loop can stop it before the test ends instead of depending on process lifetime.
- Cleanup behavior remains centered on the existing sweep logic; only lifecycle control becomes more explicit.

## Acceptance Criteria

- [x] Starting periodic cleanup returns a stop hook that callers can invoke explicitly.
- [x] The application startup path still uses periodic cleanup successfully with the updated API.
- [x] Ticker-based tests stop the cleanup loop explicitly while preserving existing cleanup behavior verification.

## QA

1. Run the cleanup-related tests.
2. Verify the periodic cleanup test still proves expired jobs and files are removed on tick.
3. Verify the test captures and calls the returned stop hook before finishing.
4. Confirm the application still builds with the updated cleanup startup API.

---

*Appended after execution.*

## Completion

**Built:** `StartCleanup` now returns an explicit stop hook that waits for the cleanup goroutine to exit, and the ticker-based tests now stop the loop they start.

**Decisions:** Used a private stop channel plus `sync.Once` so cleanup ownership stays explicit, stop remains safe to call more than once, and tests can deterministically wait for the goroutine to finish.

**Deviations:** None.

**Files:** `core/cleanup.go`, `core/cleanup_test.go`, `e2e_test.go`

**Notes for next slice:** No further slices are planned for this change set; the cleanup loop API is now explicit and test-owned where needed.
