# Slice 3: Align script jobs and tests with the lifecycle contract

## Dependencies

Slice 1: Make progress semantics explicit.

## Description

Align script jobs and shared lifecycle tests with the new contract that `done` is a status transition, not an implicit progress write. This makes the job model consistent across archive and script flows even though script generation still does not emit intermediate progress.

## Expected Behaviors Addressed

- Script jobs follow the same lifecycle rule: `done` is a status transition, not a progress mutation.

## Acceptance Criteria

- [ ] Script jobs continue to complete successfully under the new `MarkDone` behavior.
- [ ] Shared job lifecycle tests explicitly reflect that completion does not rewrite progress.
- [ ] No remaining tests rely on `MarkDone` to force progress to `100` for script or archive jobs.

## QA

1. Run script job tests and broader job lifecycle tests.
2. Confirm script jobs still produce their output file and transition to `StatusDone` successfully.
3. Confirm no test assumes `MarkDone` implicitly changes progress.
4. Confirm archive and script jobs now share the same completion semantics.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
