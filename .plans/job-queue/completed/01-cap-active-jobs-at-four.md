# Slice 1: Cap Active Jobs At Four

## Dependencies

None.

## Description

Add a shared execution limit so the service still accepts new jobs immediately, but only 4 jobs can actively run at once. Extra jobs remain pending until one of the running jobs finishes and frees a slot. This applies uniformly to zip, tarball, and script jobs.

## Expected Behaviors Addressed

- When many jobs are submitted quickly, all of them are accepted and stored, but only 4 run at once.
- Extra jobs remain `pending` until one of the 4 running jobs finishes and frees a slot.
- Zip, tarball, and script jobs all share the same 4-job limit.

## Acceptance Criteria

- [ ] Submitting more than 4 jobs still returns success for each request.
- [ ] No more than 4 jobs are in active execution at the same time.
- [ ] A queued job stays `pending` until an execution slot becomes available, then transitions into normal processing.

## QA

1. Start the service.
2. Submit at least 5 jobs in quick succession across any mix of zip, tarball, and script job types.
3. Check job status responses while they are running.
4. Confirm only 4 jobs are `processing` at once and at least 1 later job remains `pending`.
5. Wait for one running job to complete.
6. Confirm one previously pending job begins processing without being resubmitted.

---

*Appended after execution.*

## Completion

**Built:** Added a shared semaphore on `Manager` so only 4 dispatched jobs can enter execution at once, while later jobs remain pending until a slot opens.

**Decisions:** Kept the existing goroutine-per-dispatch model and wrapped execution with a small `dispatchJob` helper. Added per-job executor function fields on `Manager` so concurrency behavior can be tested deterministically without changing public behavior.

**Deviations:** Did not add shutdown cancellation yet. This slice only adds bounded execution and the tests needed to prove the fifth job waits for capacity.

**Files:** Modified `internal/service/manager.go` and `internal/service/manager_test.go`.

**Notes for next slice:** Active job limiting is now centralized in `dispatchJob`, which is the natural place to layer in manager-level cancellation and shutdown waiting behavior.
