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

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
