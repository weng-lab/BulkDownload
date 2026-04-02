# Slice 1: Admin Read Endpoints With Stored Size Metadata

## Dependencies

None.

## Description

Add the first admin API slice for reading current job state. This slice extends jobs with the stored metadata needed for operational inspection, records `input_size` and `output_size` as jobs are created and completed, and exposes `GET /admin/jobs` plus `GET /admin/jobs/{id}`. The admin reads return only non-expired jobs and order the list newest first.

## Expected Behaviors Addressed

- `GET /admin/jobs` returns all current non-expired jobs.
- `GET /admin/jobs/{id}` returns one current non-expired job.
- Both admin read endpoints return the same job fields.
- `input_size` is the sum of the sizes of all requested files for that job.
- `output_size` is the size of the file the service created for that job.
- Jobs are ordered newest first in the list response.
- Expired jobs do not appear in admin responses even before the cleanup loop removes them.

## Acceptance Criteria

- [ ] Runtime jobs include `creation_time`, `input_size`, and `output_size`.
- [ ] `input_size` is recorded during normal job creation for zip, tarball, and script jobs.
- [ ] `output_size` is recorded when zip, tarball, and script jobs successfully create their output artifact.
- [ ] `GET /admin/jobs` returns all visible jobs ordered newest first.
- [ ] `GET /admin/jobs/{id}` returns the same response shape for a single visible job.
- [ ] Expired jobs are hidden from both admin read endpoints.
- [ ] Tests cover stored size metadata and admin read behavior.

## QA

1. Start the service.
2. Create one zip job and one script job through the existing public `POST /jobs` endpoint.
3. Request `GET /admin/jobs`.
4. Confirm both jobs appear with `id`, `type`, `status`, `progress`, `files`, `input_size`, `output_size`, `creation_time`, `expires_at`, and `error`.
5. Confirm the job list is ordered newest first.
6. Request `GET /admin/jobs/{id}` for one job and confirm it returns the same fields.
7. Wait for a job to expire or use a short TTL test configuration, then confirm the expired job no longer appears in either admin read endpoint.

---

*Appended after execution.*

## Completion

**Built:** Extended runtime jobs with `creation_time`, `input_size`, and `output_size`; added admin read endpoints at `GET /admin/jobs` and `GET /admin/jobs/{id}` with newest-first ordering and expired-job filtering; covered the new metadata and admin responses in unit and end-to-end tests.

**Decisions:** Kept admin reads store-backed and filtered/sorted in the HTTP layer; computed `input_size` while validating requested files; recorded `output_size` from the created artifact file immediately before marking a job done; used a dedicated admin response shape so the public status endpoint contract stayed unchanged.

**Deviations:** None.

**Files:** Modified `api/handlers.go`, `api/handlers_test.go`, `api/router.go`, `api/types.go`, `e2e_test.go`, `internal/jobs/jobs.go`, `internal/jobs/jobs_test.go`, `internal/service/archive.go`, `internal/service/archive_test.go`, `internal/service/create_job.go`, `internal/service/create_job_test.go`, `internal/service/manager.go`, `internal/service/manager_test.go`, `internal/service/script.go`, and `internal/service/script_test.go`.

**Notes for next slice:** Admin read handlers already share a stable JSON shape and hide expired jobs before cleanup runs. Delete behavior can build on the stored `Filename` and `OutputSize` metadata now present on completed jobs.
