# Slice 1: Make progress semantics explicit

## Dependencies

None.

## Description

Make archive progress reach `100` from the progress path itself instead of relying on a forced final update, and make job completion stop rewriting progress. This slice delivers the core contract cleanup end-to-end: `Progress` reflects bytes copied, while `Status` reflects lifecycle state.

## Expected Behaviors Addressed

- Archive jobs report monotonically increasing progress based on bytes copied.
- Archive jobs can reach `100` while still in `processing` until final archive/writer close steps complete.
- When finalization succeeds, the job transitions to `done` without changing progress again.
- The progress implementation is readable in one place without needing cross-file knowledge to understand why `100` appears.

## Acceptance Criteria

- [ ] Archive progress can naturally reach `100` through the progress reporter without a special forced-final-update step.
- [ ] `MarkDone` no longer mutates job progress and only updates lifecycle status plus final metadata.
- [ ] Archive and lifecycle tests pass with the new contract, including cases where `Progress == 100` before `Status == done`.

## QA

1. Run targeted tests for `core/progress.go`, archive job behavior, and job lifecycle behavior.
2. Confirm archive progress remains monotonic and ends at `100`.
3. Confirm `MarkDone` changes status and filename without overwriting an existing progress value.
4. Confirm end-to-end archive job tests still finish with `StatusDone` and `Progress == 100`.

---

*Appended after execution.*

## Completion

**Built:** Archive progress now reaches `100` from `progressReporter.Add`, archive creation no longer forces a final progress update, and `MarkDone` now changes lifecycle state plus metadata without rewriting progress. Added coverage for the new reporter behavior and for archives reporting `100` before archive creation returns.

**Decisions:** Kept the archive lifecycle contract centered on `SetProgress` updates during work and `MarkDone` for the status transition only. Added an explicit `SetProgress(jobID, 100)` in `executeScriptJob` so the current script end-to-end behavior stays green while the dedicated script-alignment slice remains pending.

**Deviations:** Extended the slice slightly into `core/script.go` to preserve the existing script observable behavior and keep the full Go test suite passing after removing the implicit progress write from `MarkDone`.

**Files:** `.plans/progress-semantics-cleanup/completed/01-make-progress-semantics-explicit.md`, `core/archive.go`, `core/archive_test.go`, `core/jobs.go`, `core/jobs_test.go`, `core/progress.go`, `core/progress_test.go`, `core/script.go`

**Notes for next slice:** `copyWithProgress` still uses the manual read/write loop and is the remaining cleanup target for slice 2. Script jobs now set `100` explicitly before `MarkDone`; slice 3 should decide whether to keep that explicit completion update or tighten script progress semantics further.
