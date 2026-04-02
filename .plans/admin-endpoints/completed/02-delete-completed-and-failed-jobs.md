# Slice 2: Delete Completed And Failed Jobs

## Dependencies

- Slice 1: Admin Read Endpoints With Stored Size Metadata

## Description

Add the first delete path for admin job management by supporting `DELETE /admin/jobs/{id}` for non-running jobs. This slice lets the admin remove completed and failed jobs, cleans up any completed artifact file if it exists, removes the job from the in-memory store, and returns success only after deletion is actually finished.

## Expected Behaviors Addressed

- I can call `DELETE /admin/jobs/{id}` for visible non-running jobs.
- If the job has a partial or completed output file, the service removes it before returning success.
- After a successful delete, the job is gone from the store and no longer appears in admin reads.
- If deletion cannot complete cleanly, the endpoint returns an error instead of claiming success.

## Acceptance Criteria

- [ ] `DELETE /admin/jobs/{id}` succeeds for `done` jobs.
- [ ] `DELETE /admin/jobs/{id}` succeeds for `failed` jobs.
- [ ] Completed artifact files are removed before the delete endpoint returns success.
- [ ] Deleted jobs no longer appear in `GET /admin/jobs` or `GET /admin/jobs/{id}`.
- [ ] Delete requests for missing or expired jobs return not found.
- [ ] Tests cover successful deletion and cleanup for completed and failed jobs.

## QA

1. Start the service.
2. Create a zip job and wait for it to complete.
3. Confirm the completed job appears in `GET /admin/jobs`.
4. Call `DELETE /admin/jobs/{id}` for that job.
5. Confirm the delete request succeeds.
6. Confirm `GET /admin/jobs/{id}` now returns not found.
7. Confirm the artifact file is removed from the jobs directory.
8. Repeat the flow with a failed job and confirm the job is removed from admin reads after deletion.

---

*Appended after execution.*

## Completion

**Built:** Added `DELETE /admin/jobs/{id}` for visible non-running jobs, with manager-level deletion that removes any stored artifact before deleting the job from the in-memory store; covered successful deletion, not-found handling, cleanup failures, and end-to-end completed-job removal.

**Decisions:** Kept deletion coordinated in `service.Manager` so file cleanup and store removal stay atomic from the handler's perspective; returned `204 No Content` on success; treated pending and processing jobs as `409 Conflict` for now so the endpoint never claims success before slice 3 adds targeted cancellation.

**Deviations:** Added explicit conflict handling for running jobs even though this slice only required completed and failed deletion, because exposing the endpoint without a safe running-job response would be misleading.

**Files:** Modified `api/handlers.go`, `api/handlers_test.go`, `api/router.go`, `e2e_test.go`, `internal/service/manager.go`, and `internal/service/manager_test.go`.

**Notes for next slice:** Running-job deletes currently return `409` via `service.ErrDeleteJobRunning`. Slice 3 can extend `Manager.DeleteJob` to cancel per-job work, wait for completion, and then reuse the same artifact cleanup and store-removal path.
