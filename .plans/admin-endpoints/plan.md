# Admin Job Management API

## Problem

I want a lightweight admin API for operational visibility and control over jobs in this service. I need to inspect current jobs with the real fields the service already knows, and I need to be able to delete any job. If a job is still running, deleting it should stop the in-flight work, clean up any created file, and remove the job from memory. The API should keep the payload simple and focused on stored job data plus the storage metrics I care about, without inventing extra derived fields for the server to maintain.

## Solution

Add a small admin API to the existing HTTP service with endpoints to list jobs, fetch one job, and delete a job. Extend the in-memory job model with `creation_time`, `input_size`, and `output_size`. Introduce per-job cancellation tracking in the manager so a delete request can cancel a specific running job, wait for the work to stop, clean up any partial or completed output artifact, and then remove the job from the store.

## Expected Behavior

- I can request `GET /admin/jobs` and get all current non-expired jobs.
- I can request `GET /admin/jobs/{id}` and get one current non-expired job.
- Both admin read endpoints return the same job fields.
- Each admin job response includes:
  - `id`
  - `type`
  - `status`
  - `progress`
  - `files`
  - `input_size`
  - `output_size`
  - `creation_time`
  - `expires_at`
  - `error`
- `input_size` is the sum of the sizes of all requested files for that job.
- `output_size` is the size of the file the service created for that job:
  - zip archive size for zip jobs
  - tarball size for tarball jobs
  - generated script size for script jobs
- Jobs are ordered newest first in the list response.
- Expired jobs do not appear in admin responses even before the cleanup loop removes them.
- I can call `DELETE /admin/jobs/{id}` for any visible job state: `pending`, `processing`, `done`, or `failed`.
- If the job is pending or processing, the service cancels it and waits for the work to actually stop before returning success.
- If the job has a partial or completed output file, the service removes it before returning success.
- After a successful delete, the job is gone from the store and no longer appears in admin reads.
- If deletion cannot complete cleanly, the endpoint returns an error instead of claiming success.

## Implementation Decisions

- Reuse the existing HTTP server and router rather than adding a separate admin process.
- Keep the admin surface as JSON endpoints only.
- Use the same response shape for list and detail endpoints.
- Extend the runtime job state with:
  - `creation_time`
  - `input_size`
  - `output_size`
- Compute `input_size` during normal job creation while file paths are already being validated.
- Record `output_size` when a job successfully creates its output file.
- Keep the job store as the source of truth for visible runtime job state.
- Add a list operation to the job store, but keep filtering and sorting in the HTTP layer.
- Add per-job execution control in the manager so each dispatched job has its own cancellable context and completion signal.
- Treat job deletion as a manager-level operation, not a raw store mutation, because correct deletion depends on coordination with in-flight work and artifact cleanup.
- A delete request for a running job should:
  - signal cancellation for that specific job
  - wait for the job goroutine to finish
  - remove any partial or completed output file
  - remove the job from the store
- A delete request for a non-running job should:
  - remove any output file if present
  - remove the job from the store
- Job execution paths should tolerate targeted cancellation without leaving partial files behind.
- Expired jobs remain hidden from admin reads and should be treated as not found for single-job operations.
- Keep auth, CORS hardening, and network restriction work out of this slice.

## Testing Approach

- Add store tests for the new list operation, but do not add tests that are purely about pointer-vs-copy implementation details.
- Extend service-level job creation tests to verify jobs now record:
  - `creation_time`
  - `input_size`
- Extend job execution tests to verify successful jobs record `output_size` for:
  - zip jobs
  - tarball jobs
  - script jobs
- Add manager-level deletion tests as table-driven tests covering:
  - deleting pending jobs
  - deleting processing archive jobs
  - deleting processing script jobs
  - deleting done jobs with output files
  - deleting failed jobs
  - waiting for cancellation before returning
  - removing files and removing store entries
  - error behavior when cleanup fails
- Reuse the existing archive cancellation test patterns as prior art for partial-file cleanup during cancellation.
- Add handler tests for:
  - list jobs
  - get single job
  - delete job
  - newest-first ordering
  - exclusion of expired jobs
  - missing job behavior
  - correct `input_size` and `output_size` fields in JSON responses
  - exact response shape by checking expected fields and total field count
- Add end-to-end coverage that:
  - creates a job through the public API
  - verifies admin read responses
  - deletes the job through the admin API
  - confirms the job is no longer readable
  - confirms output cleanup when applicable

## Out of Scope

- HTML pages or dashboards
- A dedicated admin CLI
- Authentication or authorization
- Config flags to enable or disable the admin API
- Persisting jobs across process restarts
- Filtering, pagination, or summary stats on the admin endpoints
- Returning client-derived convenience fields
- Network-level access restriction work
- CORS tightening or origin policy changes
