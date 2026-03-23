# Progress Semantics Cleanup

## Problem

The current progress flow mixes two different meanings of completion. The progress reporter in `core/progress.go` intentionally stops at `99`, then archive code forces `100`, and `MarkDone` also sets progress to `100`. That spreads one small contract across multiple files and makes it harder to reason about what progress actually means.

At the same time, `copyWithProgress` owns a custom read/write loop even though the behavior is standard byte copying with counting. That adds code surface for something the standard library already handles well.

## Solution

Make progress mean one thing everywhere: bytes copied relative to total bytes. Let the progress reporter reach `100` naturally when copying finishes, even if the job has not been marked done yet.

Separate lifecycle state from transfer progress. `MarkDone` should only mark the job as done and set final metadata, not rewrite progress. Archive and script code should no longer depend on a hidden `99 -> forced 100 -> MarkDone` sequence.

Simplify copy tracking by delegating the actual transfer to standard I/O helpers and layering progress counting around that path.

## Expected Behavior

- Archive jobs report monotonically increasing progress based on bytes copied.
- Archive jobs can reach `100` while still in `processing` until final archive/writer close steps complete.
- When finalization succeeds, the job transitions to `done` without changing progress again.
- Script jobs follow the same lifecycle rule: `done` is a status transition, not a progress mutation.
- The progress implementation is readable in one place without needing cross-file knowledge to understand why `100` appears.

## Implementation Decisions

- Treat `Progress` as transfer progress only, independent from job lifecycle state.
- Allow the progress reporter to emit `100` when `copied >= total`; remove the artificial `99` clamp.
- Keep `StatusProcessing` and `StatusDone` as the source of truth for lifecycle stage, even if progress is already `100`.
- Change `MarkDone` so it updates status and final filename only; it should not set progress.
- Replace the manual read/write loop with `io.Copy` or `io.CopyBuffer` plus a small counting wrapper that reports bytes as they are written.
- Preserve the current monotonic progress behavior and clamping to the valid range.
- Keep script jobs consistent with the same contract, even if they still do not emit intermediate progress.

## Testing Approach

- Update unit tests around the progress reporter to assert that `100` is reachable from the reporter itself.
- Update archive tests to continue asserting monotonic progress and final `100`, but no longer rely on a special forced-final-update contract.
- Add or adjust lifecycle tests so `MarkDone` no longer changes progress.
- Keep coverage for `SetProgress` clamping behavior if that API remains public and accepts arbitrary values.
- Verify end-to-end job tests still show archive jobs finishing with `StatusDone` and `Progress == 100`, now because copying reached `100` before completion rather than because `MarkDone` overwrote it.

## Out of Scope

- Adding intermediate progress reporting for script generation beyond the current behavior.
- Redesigning the broader jobs API or persistence model.
- Changing archive byte accounting beyond the existing "sum of source file sizes" model.
