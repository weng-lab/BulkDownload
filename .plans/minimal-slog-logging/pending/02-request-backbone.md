# Slice 2: Request backbone

## Dependencies

Slice 1: Tracer logger.

## Description

Add one small HTTP middleware that derives a request-scoped logger, stores it in request context, and emits one consistent finished-request log line for every request. This creates a clean request logging backbone without changing endpoint behavior.

## Expected Behaviors Addressed

- Every HTTP request emits one consistent finished-request log line with status and duration.

## Acceptance Criteria

- [ ] Every routed HTTP request gets a request-scoped logger in context.
- [ ] Every request emits exactly one finished-request log with stable fields such as `method`, `path`, `status`, and `duration_ms`.
- [ ] The middleware stays small and does not introduce a broad logging abstraction.

## QA

1. Start the service.
2. Send a successful request to `POST /jobs`.
3. Send a request to `GET /status/{id}` for an unknown job.
4. Confirm each request produces one `request finished` log line with the correct method, path, status, and duration.

---

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
