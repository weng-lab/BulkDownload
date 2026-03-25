# Slice 1: Add baseline CI workflow

## Dependencies

None.

## Description

Add a GitHub Actions CI workflow that runs on pull requests and pushes to `main`, sets up Go, runs the repository's test suite, and verifies the application still builds cleanly. This gives the repo an end-to-end validation path for normal development work without touching release automation yet.

## Expected Behaviors Addressed

- When a pull request is opened or updated, GitHub Actions runs CI automatically and reports whether the Go project passes validation.
- When code is pushed to `main`, GitHub Actions reruns the same CI checks so the main branch stays healthy.

## Acceptance Criteria

- [ ] A GitHub Actions workflow exists for pull requests and pushes to `main`.
- [ ] The workflow sets up the repository's Go toolchain and runs the full Go test suite.
- [ ] The workflow verifies the application still builds cleanly from source.
- [ ] A failing test or build step causes the workflow to fail clearly.

## QA

1. Review the CI workflow file and confirm it triggers on pull requests and pushes to `main`.
2. Review the workflow steps and confirm it installs Go, runs `go test ./...`, and verifies the application build.
3. Push a small test branch or open a small pull request and confirm the CI workflow starts automatically.
4. Confirm the workflow reports success when tests and build pass.
5. Intentionally break a test or build in a temporary branch and confirm the workflow fails.

---

## Completion

**Built:** Added `.github/workflows/ci.yml` to run GitHub Actions CI on pull requests and pushes to `main`, with `go test ./...` followed by `go build .`.

**Decisions:** Used `actions/setup-go` with `go-version-file: go.mod` so CI follows the repository's declared Go toolchain, and kept the workflow to a single test-and-build job for a straightforward baseline validation path.

**Deviations:** Did not execute the GitHub-hosted trigger checks from the QA list locally because those require pushing a branch or opening a pull request in GitHub; local verification covered the workflow contents plus successful `go test ./...` and `go build .` execution.

**Files:** Added `.github/workflows/ci.yml`; updated and moved this slice record to `.plans/github-actions-cicd/completed/01-add-baseline-ci-workflow.md`.

**Notes for next slice:** Release automation can assume baseline CI now uses the repo Go version and validates the app with the same `go test ./...` and `go build .` commands that should be reused before Docker publishing.
