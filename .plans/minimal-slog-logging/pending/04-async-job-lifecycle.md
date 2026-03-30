# Slice 4: Async job lifecycle

## Dependencies

Slice 1: Tracer logger.

## Description

Log async job execution through the shared logger with job-scoped fields so background work remains readable and easy to correlate. Keep the logging limited to major lifecycle events: job started, job completed, and job failed.

## Expected Behaviors Addressed

- Each async job logs its own lifecycle independently, including start, completion, and failure, tied together by `job_id`.

## Acceptance Criteria

- [ ] Zip, tarball, and script jobs log a clear start event.
- [ ] Successful jobs log a clear completion event.
- [ ] Failed jobs log a clear error event with `job_id` and `job_type`.

## QA

1. Start the service.
2. Create one successful job and follow the logs until completion.
3. Trigger one failing job path if practical and follow the logs until failure.
4. Confirm the async logs are easy to correlate by `job_id` and show only the main lifecycle steps.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
