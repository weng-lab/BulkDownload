# Minimal `slog` Logging

## Problem

I want the service logs to explain what is happening without turning stdout into noise. The current logging is scattered, unstructured, and incomplete. I want a simple `slog` setup with readable output, a few clear log levels, and logs at the major steps for each endpoint and job lifecycle.

## Solution

Replace direct stdlib `log` usage with a single shared `slog` logger. Derive a request-scoped logger in HTTP middleware, then use that logger in handlers for high-level request flow. Use the base logger for service startup/shutdown and for background job execution. Keep messages human-first and short, while attaching structured fields for parsing and correlation.

## Expected Behavior

- When the service starts, it logs its configuration at a readable high level and reports that it is listening.
- Every HTTP request emits one consistent finished-request log line with status and duration.
- Each endpoint logs only the major steps and outcomes that matter, such as job accepted, job not found, job not ready, or download served.
- Each async job logs its own lifecycle independently, including start, completion, and failure, tied together by `job_id`.
- Cleanup emits a debug log every sweep, an info log when it removes expired jobs, and error logs when cleanup work fails.
- Changing `LOG_LEVEL` adjusts verbosity without code changes.

## Implementation Decisions

- Use `log/slog` from the standard library rather than a third-party logger.
- Keep a single base logger for the process and pass or derive child loggers rather than importing global logging directly across packages.
- Use text output, not JSON, so local stdout stays easy to scan.
- Prefer human-readable messages with consistent verbs and nouns, while keeping machine-parseable fields attached.
- Introduce a small request-logger middleware that stores the logger in request context and records request completion details.
- Use context lookup in handlers for request logging, and base-logger-derived logging for async job execution.
- Correlate async job and request activity through `job_id` and `job_type`; do not add request IDs in this pass.
- Keep default log level at `INFO` and support a simple env-driven override such as `LOG_LEVEL=debug`.
- Log only high-level service flow, notable state transitions, and meaningful errors; do not try to log every function call.
- Preserve readability over abstraction: small explicit logging calls at important boundaries are preferred.

## Testing Approach

- Add tests for log-level parsing and default logger configuration behavior.
- Add middleware tests that verify the request-finished log is emitted with the expected fields and status handling.
- Add handler-focused tests where needed to confirm important branches still behave correctly after logger injection or context lookup.
- Add or adjust job execution tests to verify lifecycle logging hooks do not change existing status/progress behavior.
- Add cleanup tests to cover debug sweep logging and info/error behavior when expired jobs are processed.
- Favor targeted tests around behavior and integration points rather than snapshotting every exact log line.

## Out of Scope

- Request IDs or distributed tracing.
- Broad debug-level instrumentation across all helpers and internal functions.
- JSON log output or external log sinks.
- A complex logging package with many wrappers or custom abstractions.
- Changes to business behavior unrelated to logging.
