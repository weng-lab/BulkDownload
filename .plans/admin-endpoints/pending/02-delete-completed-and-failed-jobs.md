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

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
