# Job Queue

## Problem

We currently start every job immediately in its own goroutine. That means the service can end up doing an unbounded amount of file copying and archive creation at once, which is risky for disk I/O, CPU, and shutdown behavior. On top of that, shutdown does not currently stop in-flight work, so we can leave partial `.zip`, `.tar.gz`, or `.sh` artifacts behind.

## Solution

Add a manager-owned execution gate that allows at most 4 jobs to run concurrently, regardless of job type. Jobs can still be created and stored immediately as `pending`, but only 4 worker goroutines are allowed to enter actual execution at a time.

At the same time, thread cancellation through job execution so shutdown can signal all active work to stop. When a job is interrupted during shutdown, treat that as a cancellation path: stop copying/writing, delete the partial output file, and mark the job as failed with a cancellation error.

## Expected Behavior

- When many jobs are submitted quickly, all of them are accepted and stored, but only 4 run at once.
- Extra jobs remain `pending` until one of the 4 running jobs finishes and frees a slot.
- Zip, tarball, and script jobs all share the same 4-job limit.
- On normal job failure, partial output files are still removed.
- On service shutdown, active jobs stop as soon as they hit the next cancellation check.
- If shutdown interrupts a job, its half-written artifact is deleted and the job does not appear as a completed download.
- Pending jobs that never started can remain in the store as `pending`; they will naturally age out via existing cleanup unless we decide to explicitly fail them later.

## Implementation Decisions

- Use a buffered semaphore on `Manager` for concurrency limiting because it matches the current goroutine-per-job structure without introducing a larger worker-pool abstraction.
- Also give `Manager` a root `context.Context` plus `cancel` function so shutdown can broadcast cancellation to all running jobs.
- Wrap the common dispatch goroutine setup in one helper so `DispatchZipJob`, `DispatchTarballJob`, and `DispatchScriptJob` all use the same acquire/run/release flow.
- Acquire the semaphore inside the goroutine, then check for manager cancellation both before starting work and during artifact generation. That preserves current request behavior: dispatch returns immediately, but execution is bounded.
- Change execution methods to accept a context so they can stop cleanly during shutdown.
- Make archive copy loops context-aware rather than relying on `io.CopyBuffer` alone; that gives us periodic cancellation checks during large file copies.
- Make script creation context-aware too, even though it is fast, so all job types behave consistently under shutdown.
- Keep cleanup explicit: on any execution error, including cancellation, remove the output file and mark the job failed.
- Add a manager shutdown method that cancels active work and waits for launched job goroutines to exit. `main.go` should call that during shutdown after stopping intake from HTTP.

## Testing Approach

- Add manager tests that submit more than 4 jobs and verify the 5th stays `pending` until a slot is released.
- Add execution tests proving a cancelled zip/tarball job removes its partial artifact and ends in `failed`.
- Add a shutdown-oriented manager test that starts long-running jobs, calls manager shutdown, and verifies goroutines exit and partial files are cleaned up.
- Add lower-level artifact tests for context-aware copy behavior so cancellation during file copy is deterministic and does not depend on timing luck.
- Keep existing success-path tests for done/filename/progress behavior and update them only where signatures change.

## Out of Scope

- Changing the HTTP API shape or exposing queue depth/concurrency configuration to users.
- Introducing a full worker-pool or persistent job queue.
- Distinguishing cancelled jobs from failed jobs with a new public job status.
- Failing never-started pending jobs during shutdown.
