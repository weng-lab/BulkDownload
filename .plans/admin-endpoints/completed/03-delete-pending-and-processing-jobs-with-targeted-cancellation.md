# Slice 3: Delete Pending And Processing Jobs With Targeted Cancellation

## Dependencies

- Slice 2: Delete Completed And Failed Jobs

## Description

Extend admin deletion to active work by adding targeted per-job cancellation in the manager. This slice allows `DELETE /admin/jobs/{id}` to stop pending and processing jobs, wait for the job goroutine to finish, clean up any partial or completed output artifact, and only then remove the job from the store.

## Expected Behaviors Addressed

- I can call `DELETE /admin/jobs/{id}` for visible pending and processing jobs.
- If the job is pending or processing, the service cancels it and waits for the work to actually stop before returning success.
- If the job has a partial output file, the service removes it before returning success.
- After a successful delete, the job is gone from the store and no longer appears in admin reads.
- If deletion cannot complete cleanly, the endpoint returns an error instead of claiming success.

## Acceptance Criteria

- [ ] The manager tracks per-job cancellation and completion state.
- [ ] `DELETE /admin/jobs/{id}` succeeds for pending jobs.
- [ ] `DELETE /admin/jobs/{id}` succeeds for processing zip or tarball jobs and waits for work to stop.
- [ ] `DELETE /admin/jobs/{id}` succeeds for processing script jobs and waits for work to stop.
- [ ] Partial artifacts are removed before the delete endpoint returns success.
- [ ] Deleted jobs no longer appear in `GET /admin/jobs` or `GET /admin/jobs/{id}`.
- [ ] Table-driven tests cover deletion outcomes across active job states.

## QA

1. Start the service with a job large enough to stay in `processing` briefly.
2. Create a zip or tarball job.
3. While the job is pending or processing, call `DELETE /admin/jobs/{id}`.
4. Confirm the delete request does not return until the job has actually stopped.
5. Confirm `GET /admin/jobs/{id}` now returns not found.
6. Confirm no partial artifact remains in the jobs directory.
7. Repeat with another active job type if needed to verify cancellation behavior across job implementations.

---

*Appended after execution.*

## Completion

**Built:** Added per-job cancellation and completion tracking in `service.Manager`, so `DELETE /admin/jobs/{id}` now cancels pending and processing jobs, waits for the specific job goroutine to stop, removes any derived artifact path, and then deletes the job from the store.

**Decisions:** Kept the existing global manager shutdown context and layered per-job `context.WithCancel` state on top of it rather than refactoring execution flow more broadly; reused the delete path for active and non-active jobs by deriving the artifact filename from job type when a running job has not yet stored `Filename`; verified delete blocking behavior with manager-level tests that control the active run lifecycle directly.

**Deviations:** Did not add a new HTTP handler code path because the existing delete handler already delegates to `Manager.DeleteJob`; once active-job deletion became safe at the manager layer, the endpoint behavior updated without extra transport changes.

**Files:** Modified `internal/service/manager.go` and `internal/service/manager_test.go`.

**Notes for next slice:** Active-job delete no longer returns `service.ErrDeleteJobRunning` in normal manager-driven flows, but the handler still maps that error if it ever surfaces from an unexpected edge case. Full-suite verification passed with `go test ./...`.
