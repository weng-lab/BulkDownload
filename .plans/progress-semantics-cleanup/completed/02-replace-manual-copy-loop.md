# Slice 2: Replace manual copy loop

## Dependencies

Slice 1: Make progress semantics explicit.

## Description

Refactor the progress-aware copy path to use standard library copy helpers with a small counting adapter instead of a custom read/write loop. This keeps the behavior delivered in slice 1 while simplifying the implementation and making the progress path easier to reason about.

## Expected Behaviors Addressed

- The progress implementation is readable in one place without needing cross-file knowledge to understand why `100` appears.
- Archive jobs report monotonically increasing progress based on bytes copied.

## Acceptance Criteria

- [ ] `copyWithProgress` delegates byte transfer to `io.Copy` or `io.CopyBuffer` instead of owning a manual read/write loop.
- [ ] Progress reporting remains monotonic and preserves the same observable archive behavior as slice 1.
- [ ] Archive content and completion tests pass without regressions after the refactor.

## QA

1. Run the same archive and progress tests used for slice 1.
2. Confirm generated zip and tarball outputs are unchanged from the caller's perspective.
3. Confirm progress still advances correctly during copy and ends at `100`.
4. If a new counting helper exists, run its focused unit test as well.

---

*Appended after execution.*

## Completion

**Built:** `copyWithProgress` now delegates byte transfer to `io.CopyBuffer` through a small `progressWriter` adapter, and added a focused unit test covering copied output plus progress reporting.

**Decisions:** Count progress from bytes successfully written to the destination instead of bytes merely read, so the reporting contract stays aligned with completed transfer work while letting `io.CopyBuffer` own the copy loop.

**Deviations:** None.

**Files:** `.plans/progress-semantics-cleanup/completed/02-replace-manual-copy-loop.md`, `core/progress.go`, `core/progress_test.go`

**Notes for next slice:** Script jobs still set progress to `100` explicitly before `MarkDone`; slice 3 can now focus on whether that behavior should remain or be reshaped without any archive copy-loop cleanup left.
