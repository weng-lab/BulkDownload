# Slice 3: Job create flow

## Dependencies

Slice 2: Request backbone.

## Description

Use the request-scoped logger in the create, status, and download handlers to log only the important high-level branches. This makes each endpoint easy to follow in stdout without adding noisy helper-level logs.

## Expected Behaviors Addressed

- Each endpoint logs only the major steps and outcomes that matter, such as job accepted, job not found, job not ready, or download served.

## Acceptance Criteria

- [ ] The create handler logs high-level outcomes such as accepted job and failed dispatch.
- [ ] The status and download handlers log meaningful branches such as not found, not ready, and served download.
- [ ] Endpoint logs use human-readable messages plus structured fields like `job_id` and `job_type` where relevant.

## QA

1. Start the service.
2. Create a valid job and confirm the handler logs that the job was accepted.
3. Request status for a missing job and confirm the handler logs that the job was not found.
4. Request a download before a job is ready and confirm the handler logs that the job is not ready.
5. Request a finished download and confirm the handler logs that the download was served.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
