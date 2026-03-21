## Aggregated Go Testing Analysis — core/ Package

### archive_test.go

**GOOD:**
- Proper `TestUnit_Condition` naming convention
- Correct use of `t.TempDir()` for auto-cleanup
- Proper `t.Fatalf` for setup errors
- Good test helper pattern with `testManager()`
- Internal package structure for unexported functions
- Real archive validation (reads back contents)

**NEEDS CHANGE:**
- Hand-rolled comparisons instead of `cmp.Diff` (lines 61-69, 102, 132, 200-208)
- Not table-driven (zip and tarball tests are nearly identical)
- Missing `t.Parallel()` for independent tests
- Hand-rolled time comparison (lines 89-90) instead of `cmpopts.EquateApprox`
- No subtests for complex scenarios
- Insufficient edge case coverage (empty file list, duplicate basenames)

---

### config_test.go

**GOOD:**
- Proper table-driven tests with `tests` slice and `tt` loop var
- Uses `cmp.Diff` for struct comparisons (not hand-rolled)
- Good subtest naming with descriptive `t.Run()`
- Correct helper design (`clearConfigEnv` accepts `*testing.T` first, calls `t.Helper()`)
- Uses `t.Setenv()` for env var cleanup
- Correctly avoids `t.Parallel()` for env-manipulating tests

**NEEDS CHANGE:**
- Error assertion uses `bool` instead of checking specific error type with `errors.Is()`
- Missing edge case: empty string env var behavior
- Test name `TestLoadConfig` could be more specific (`TestLoadConfig_FromEnv`)
- Should split happy/error tables (has `wantErr` field only for 2-3 cases)
- `clearConfigEnv` should be in `helpers_test.go`
- Missing edge cases: zero/negative duration, whitespace in env values

---

### script_test.go

**GOOD:**
- Internal test package (`package core`)
- Test helpers in `helpers_test.go` with proper `t.Helper()` calls
- Good `TestUnit_Condition` naming (e.g., `TestCreateDownloadScriptWritesExpectedContent`)
- Proper `t.TempDir()` usage
- Correct `t.Fatalf` for setup failures (file operations)

**NEEDS CHANGE:**
- Hand-rolled string checks with `strings.Contains` instead of `cmp.Diff` or golden files
- `t.Fatalf` used for assertion failures (lines 34-36, 43-45, 67-68) — should use `t.Errorf`
- No subtests with `t.Run` for individual checks
- Missing `t.Parallel()` for independent tests
- Magic number `0o755` without context
- `%v` instead of `%q` for string content comparisons

---

### cleanup_test.go

**GOOD:**
- Uses `testManager` helper for centralized setup
- Appropriate `t.Fatalf` for setup errors
- Specific error checking with `os.IsNotExist()` (not string matching)
- Focused test scope (single responsibility per test)
- Edge case coverage (missing file scenario)

**NEEDS CHANGE:**
- Not table-driven despite similar structure across tests
- Hand-rolled assertions instead of `cmp.Diff`
- Missing `t.Parallel()` for independent tests
- Test naming uses camelCase instead of snake_case with underscore
- Missing `StartCleanup` tests (zero coverage for goroutine/ticker)
- No subtests for multi-assertion scenarios
- No explicit cleanup for test-created files

---

### jobs_test.go

**GOOD:**
- Uses `cmp.Diff` consistently for struct comparisons
- Uses `errors.Is` for error checking (proper sentinel errors)
- Correct `t.Fatalf` vs `t.Errorf` usage
- Tests internal state isolation (snapshot behavior)
- Clear `TestUnit_Condition` naming
- One `_test.go` per source file convention

**NEEDS CHANGE:**
- Not table-driven (multiple functions could be consolidated)
- Missing `t.Parallel()` for independent tests
- `TestJobsAddGetDelete` tests too much (Add, Get, Delete, AND state isolation)
- Incomplete edge case coverage (nil job, empty ID)
- Should split into focused tests or use subtests

---

### helpers_test.go

**GOOD:**
- Helper functions accept `*testing.T` first and call `t.Helper()` correctly
- Follows `testXxx` naming convention
- Uses `t.TempDir()` for `JobsDir` cleanup
- Reusable across package tests

**NEEDS CHANGE:**
- `testManager` returns too many values (causes caller confusion)
- Missing `t.Cleanup()` for manager lifecycle (goroutines, file handles)
- Hardcoded `Port: "8080"` could conflict in parallel tests

---

### manager_test.go

**GOOD:**
- Dependency injection for ID generation (deterministic retry testing)
- Proper `t.Fatalf` vs `t.Errorf` distinction
- Internal package testing of unexported functions
- Clear, single-purpose test focus

**NEEDS CHANGE:**
- Not table-driven (two similar tests should be consolidated with `t.Run`)
- Hand-rolled assertions instead of `cmp.Diff` (lines 26-28, 44-45)
- Missing error type check on exhaustion test (just checks `err != nil`)
- No baseline success test (happy path without collisions)
- Missing `t.Parallel()` for independent tests
- Test naming verbose — prefer `TestCreateZipJob_RetriesDuplicateID`
- Incomplete coverage: `CreateTarballJob` and `CreateScriptJob` not tested
- No test for `maxJobIDAttempts` boundary
