# Slice 2: Extract Runtime Config

## Dependencies

None.

## Description

Move environment loading, default values, and config parsing into `internal/config`, then rewire bootstrap and tests so startup behavior stays the same while runtime concerns become easier to find.

## Expected Behaviors Addressed

- `main.go` still wires the app together, but package names make the app easier to scan.
- The service keeps the same external behavior: same routes, same job lifecycle, same artifact outputs.

## Acceptance Criteria

- [x] Config loading and validation live in `internal/config`.
- [x] `main.go` and relevant tests use `internal/config` instead of `core` for runtime configuration.
- [x] Default values, env overrides, and invalid config handling remain unchanged.

## QA

1. Run the config tests and confirm defaults, env overrides, and invalid duration handling still pass.
2. Run a representative startup path through API or end-to-end tests.
3. Confirm production code no longer imports `core` for config loading.

---

*Appended after execution.*

## Completion

**Built:** Moved runtime config loading, env merging, defaults, and validation into `internal/config`, then rewired bootstrap, handlers, and tests to consume the new package without changing startup or request behavior.

**Decisions:** Moved the `Config` type itself into `internal/config` so runtime configuration has a single home, and updated `core.Manager` plus API entry points to accept that concrete type directly instead of keeping a temporary alias in `core`.

**Deviations:** No functional deviations; instead of leaving a compatibility shim behind, the slice updated the remaining callers directly because the config surface was small and test coverage already exercised the integration points.

**Files:** Created `internal/config/config.go` and `internal/config/config_test.go`; removed `core/config.go` and `core/config_test.go`; updated `main.go`, `api/handlers.go`, `api/handlers_test.go`, `e2e_test.go`, `core/helpers_test.go`, `core/manager.go`, and `core/manager_test.go`.

**Notes for next slice:** Artifact generation code still lives in `core`; the next slice can move archive, script, progress, and artifact file helpers into `internal/artifacts` without needing to touch runtime config again.
