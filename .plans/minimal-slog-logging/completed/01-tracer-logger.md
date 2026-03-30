# Slice 1: Tracer logger

## Dependencies

None.

## Description

Set up one shared `slog` logger for the service with human-readable text output, a default `INFO` level, and a simple `LOG_LEVEL` override. Use it for startup and shutdown so the app has one clear logging path end-to-end.

## Expected Behaviors Addressed

- When the service starts, it logs its configuration at a readable high level and reports that it is listening.
- Changing `LOG_LEVEL` adjusts verbosity without code changes.

## Acceptance Criteria

- [x] The service creates and uses a shared `slog` logger instead of direct top-level stdlib `log` calls.
- [x] `LOG_LEVEL` supports a simple override such as `debug` while defaulting to `INFO`.
- [x] Startup and shutdown logs are emitted through `slog` in a readable text format.

## QA

1. Start the service normally.
2. Confirm startup logs are readable text and include the service starting to listen.
3. Stop the service and confirm shutdown logs are emitted through the same logger.
4. Start the service with `LOG_LEVEL=debug` and confirm debug-level logging is enabled.

---

*Appended after execution.*

## Completion

**Built:** Added a shared `slog` text logger for the service, replaced the top-level stdlib `log` usage in `main`, and added `LOG_LEVEL` config support with a default of `info`.

**Decisions:** Kept the setup simple and local to the entrypoint. Added a small `parseLogLevel` helper plus `newLogger` constructor in the `main` package instead of introducing a logging package or wrapper layer. Accepted `debug`, `info`, `warn`, and `error` as the supported levels.

**Deviations:** Did not add any debug log statements yet. Slice 1 only establishes level handling and the shared logger path; later slices can emit debug logs where they matter.

**Files:** Modified `main.go`, `internal/config/config.go`, and `internal/config/config_test.go`. Added `logging.go` and `logging_test.go`.

**Notes for next slice:** Request middleware can now derive request-scoped loggers from the shared `slog` setup. `appconfig.Config` now carries `LogLevel`, so later slices can use the configured logger without extra env reads.
