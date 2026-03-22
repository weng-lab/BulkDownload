# Simplification Opportunities

## `api/handlers.go` - DONE

- The three create handlers are near copy-paste (`zip`, `tarball`, `script`): same decode/create/log/spawn/respond flow with tiny differences. Good target for one shared `handleCreateJob` path.
- `HandleStatus` and `HandleDownload` parse IDs by trimming `r.URL.Path`; that is brittle and surprising when the router already owns path params.
- `httpStatusForCreateError` is effectively dead abstraction right now; every branch returns `400`.
- Path validation is split across `normalizeRelativePath`, `resolveArchivePath`, and `resolveArchiveFiles`, with different rules for scripts vs archives and special behavior when `SOURCE_ROOT_DIR` is empty.

## `core/archive.go`

- `ProcessZipJob` and `ProcessTarballJob` are almost identical end-to-end; same lifecycle, same cleanup, same progress wiring.
- `createZipFromRoot` and `createTarballFromRoot` repeat the same pipeline: validate, map inputs, compute size, build reporter, create output, loop files, finalize progress.
- Thin wrappers like `createZip`, `createTarball`, and `addFileToTarball` add indirection without much value.
- `archiveInputList` plus `sourcePaths()` feels like extra type ceremony for a short-lived transformation.
- validate files should os.Stat to ensure they exist and not check for abs path. 

## `core/script.go` - SKIP (SUBJECT TO CHANGE)

- Script generation uses `text/template` plus `scriptTemplateData`, `scriptTemplateFile`, and `LineSuffix` just to print a shell list; that abstraction is heavier than the problem.
- Quoting and formatting are spread across Go structs, template syntax, and bash behavior, which makes small changes harder than they should be.
- `ProcessScriptJob` duplicates the same job lifecycle pattern used in archive processing.

## `core/manager.go`

- `Manager` carries several thin state helpers (`setStatus`, `setProgress`, `setFailed`, `setDone`) that mostly wrap `Jobs.Update`; useful, but currently a lot of surface area for simple mutations.
- `CreateZipJob`, `CreateTarballJob`, and `CreateScriptJob` are pass-throughs to one internal method.
- `createJob` does `Add` and then `Get` immediately after, which suggests the storage API is a little awkward.

## `core/jobs.go`

- The store mixes two models: `Add`/`Get` return snapshots, while `Update` mutates the live stored pointer under lock. That inconsistency increases cognitive load.
- `core.Job` is both internal storage and API response model via JSON tags, which couples internals to HTTP output.

## `core/config.go`

- `LoadConfig()` mutates process-global env by reading `.env` and calling `os.Setenv`; surprising side effect for a config loader and awkward in tests.
- Config semantics are inconsistent: empty string means fallback, whitespace handling differs by type, and zero/negative durations are accepted.
- Supporting both `JOB_TTL` and `ZIP_TTL` adds compatibility branching in the core path.

## `core/progress.go`

- Progress intentionally clamps at `99%`, then archive/script code forces `100%` elsewhere; that cross-file contract is weird for such a small feature.
- `copyWithProgress` reimplements a manual read/write loop that could likely be simpler with standard I/O helpers plus counting.

## `core/cleanup.go`

- `StartCleanup` launches a ticker goroutine with no stop mechanism or context, so lifecycle ownership is implicit and test-unfriendly.

## `e2e_test.go`

- `newTestApp()` recreates route wiring instead of sharing app construction with `main.go`, so test and production setup can drift.
- Multiple tests rely on polling loops and timing windows, which adds flake risk and hides the actual behavior under retry logic.

## `api/handlers_test.go`

- There is a growing mini test framework here (`handlerFixture`, file writers, polling helper), which is useful but also a sign the production boundaries may be harder to test than necessary.
- HTTP contract tests and async job-completion behavior are mixed together more than they need to be.

## `core/script_test.go` - SKIP (SUBJECT TO CHANGE)

- `expectedDownloadScript()` mirrors the production script almost line-for-line, so harmless formatting changes create large noisy test diffs.
- This is effectively a golden test embedded in Go code, without the simplicity of a real golden file.

## `core/archive_test.go`

- This file is doing a lot at once: archive creation, manager lifecycle, progress, readers, fixtures, helpers.
- Zip and tarball behavior is re-verified in several slightly different layers, which feels heavier than necessary.

## `README.md`

- The docs appear to carry complexity from the codebase: multiple helper scripts, mixed path conventions, and more than one "local workflow" story.

## `scripts/request.sh`

- Feels inconsistent with the other scripts: odd conventions, hardcoded localhost, and a different style from the archive request helpers.
