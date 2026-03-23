# Slice 3: Align script jobs and tests with the lifecycle contract

## Dependencies

Slice 1: Make progress semantics explicit.

## Description

Align script jobs and shared lifecycle tests with the new contract that `done` is a status transition, not an implicit progress write. This makes the job model consistent across archive and script flows even though script generation still does not emit intermediate progress.

## Expected Behaviors Addressed

- Script jobs follow the same lifecycle rule: `done` is a status transition, not a progress mutation.

## Acceptance Criteria

- [x] Script jobs continue to complete successfully under the new `MarkDone` behavior.
- [x] Shared job lifecycle tests explicitly reflect that completion does not rewrite progress.
- [x] No remaining tests rely on `MarkDone` to force progress to `100` for script or archive jobs.

## QA

1. Run script job tests and broader job lifecycle tests.
2. Confirm script jobs still produce their output file and transition to `StatusDone` successfully.
3. Confirm no test assumes `MarkDone` implicitly changes progress.
4. Confirm archive and script jobs now share the same completion semantics.

---

*Appended after execution.*

## Completion

**Built:** Script jobs now finish by transitioning status to `done` without forcing `Progress` to `100`, and added focused coverage for script execution plus end-to-end assertions that script jobs complete with unchanged progress.

**Decisions:** Kept script progress at `0` because script generation still has no transfer-progress updates; lifecycle completion remains a pure status/filename transition shared with archive jobs.

**Deviations:** None.

**Files:** `.plans/progress-semantics-cleanup/completed/03-align-script-jobs-and-tests.md`, `core/script.go`, `core/script_test.go`, `e2e_test.go`

**Notes for next slice:** No pending slices remain in `progress-semantics-cleanup`; the plan's progress/lifecycle contract is now aligned across archive and script jobs.
