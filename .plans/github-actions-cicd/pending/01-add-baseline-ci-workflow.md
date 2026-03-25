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

*Appended after execution.*

## Completion

What was built. Key decisions made during implementation. Any deviations from the slice plan and why. Files created or modified. Anything the next slice should be aware of.
