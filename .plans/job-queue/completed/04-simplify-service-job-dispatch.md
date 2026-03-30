# Slice 4: Simplify Service Job Dispatch

## Dependencies

Slice 2: Shutdown Cancels Running Jobs And Cleans Partial Outputs.

## Description

Remove the now-redundant per-type dispatch entry points in the service layer and consolidate job creation plus async dispatch into one service path. Keep the HTTP handler thin and continue treating job orchestration as service logic rather than moving it into the API layer.

## Expected Behaviors Addressed

- Job creation still accepts zip, tarball, and script requests through the same API.
- The service continues to validate requested files and create pending jobs before dispatching them.
- Queueing and shutdown behavior continue to apply uniformly to all job types.
- Internal service code is simpler and avoids repetitive dispatch wrappers.

## Acceptance Criteria

- [ ] `CreateJob` remains the main service entry point for job submission.
- [ ] Redundant per-type dispatch wrappers are removed or consolidated into a single internal path.
- [ ] Existing create-job behavior remains unchanged for all supported job types.

## QA

1. Start the service.
2. Submit one zip job, one tarball job, and one script job through the existing `/jobs` API.
3. Confirm each request is accepted and returns a job ID.
4. Check status for each job and confirm they proceed through the expected lifecycle.
5. Confirm there is no API behavior change while the service internals are simpler.

---

*Appended after execution.*

## Completion

**Built:** Removed the per-type dispatch wrappers from `Manager` and consolidated job submission into a single internal `createAndDispatchJob` path, while keeping `CreateJob` as the public service entry point.

**Decisions:** Kept job type parsing and file validation in `CreateJob`, then handed off to one shared create-and-dispatch helper. Left the HTTP handler unchanged so orchestration stays in the service layer rather than leaking into transport code.

**Deviations:** Removed the redundant wrapper methods entirely instead of merely consolidating their internals. The rest of the service still uses explicit `switch job.Type` execution so the code stays straightforward.

**Files:** Modified `internal/service/manager.go`, `internal/service/create_job.go`, `internal/service/manager_test.go`, and `internal/service/create_job_test.go`.

**Notes for next slice:** There are no pending slices left in `job-queue`. The service entry point for job submission is now `CreateJob`, with `createAndDispatchJob` as the single internal path for queueing work.
